package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"DaveChat/internal/config"
	"DaveChat/internal/database"
	"DaveChat/internal/handlers"
	authmiddleware "DaveChat/internal/middleware"
	ws "DaveChat/internal/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

//go:embed all:client/dist
var embedFS embed.FS

func main() {
	cfg := config.Load()

	db, err := database.ConnectDB(cfg.DBDSN)
	if err != nil {
		slog.Error("No se pudo conectar a MariaDB", "error", err)
	} else {
		defer database.CloseDB()
	}

	hub := ws.NewHub(db)
	hub.ResetPresence()

	e := echo.New()
	e.HideBanner = true

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		message := "Internal Server Error"
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			message = fmt.Sprintf("%v", he.Message)
		}
		c.JSON(code, handlers.ErrorResponse{
			Error: handlers.ErrorBody{
				Code:    http.StatusText(code),
				Message: message,
			},
		})
	}

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	store := middleware.NewRateLimiterMemoryStore(rate.Limit(100))
	e.Use(middleware.RateLimiter(store))

	if cfg.DevMode {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{cfg.AllowedOrigins},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	api := e.Group("/api")
	authHandler := &handlers.AuthHandler{DB: db, Config: cfg}
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	protected := api.Group("")
	protected.Use(authmiddleware.AuthMiddleware(cfg.JWTSecret))

	protected.GET("/auth/me", authHandler.Me)

	profileHandler := &handlers.ProfileHandler{DB: db}
	protected.GET("/profiles", profileHandler.List)
	protected.GET("/profiles/:id", profileHandler.Get)

	convHandler := &handlers.ConversationHandler{DB: db}
	protected.GET("/conversations", convHandler.List)
	protected.POST("/conversations", convHandler.Create)
	protected.GET("/conversations/:id", convHandler.Get)

	msgHandler := &handlers.MessageHandler{DB: db}
	protected.GET("/conversations/:id/messages", msgHandler.List)
	protected.POST("/conversations/:id/messages", msgHandler.Create)
	protected.GET("/conversations/:id/messages/search", msgHandler.Search)
	protected.POST("/conversations/:id/read", msgHandler.MarkRead)

	callLogHandler := &handlers.CallLogHandler{DB: db}
	protected.GET("/call-logs", callLogHandler.List)
	protected.POST("/call-logs", callLogHandler.Create)

	protected.GET("/ws", func(c echo.Context) error {
		return ws.HandleWebSocket(hub, c)
	})

	if !cfg.DevMode {
		subFS, err := fs.Sub(embedFS, "client/dist")
		if err != nil {
			e.Logger.Fatal("Failed to get embedded FS:", err)
		}

		e.GET("/assets/*", echo.WrapHandler(http.FileServer(http.FS(subFS))))

		e.GET("/*", func(c echo.Context) error {
			data, err := fs.ReadFile(subFS, "index.html")
			if err != nil {
				return c.NoContent(http.StatusNotFound)
			}
			return c.HTMLBlob(http.StatusOK, data)
		})
	}

	slog.Info("🚀 DaveChat API iniciado", "port", cfg.Port, "url", "http://0.0.0.0:"+cfg.Port)

	go func() {
		if err := e.Start("0.0.0.0:" + cfg.Port); err != nil && err != http.ErrServerClosed {
			slog.Error("Error iniciando servidor", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Server exited cleanly")
}
