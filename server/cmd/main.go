package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"DaveChat/internal/config"
	"DaveChat/internal/database"
	"DaveChat/internal/handlers"
	authmiddleware "DaveChat/internal/middleware"
	ws "DaveChat/internal/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed all:client/dist
var embedFS embed.FS

func main() {
	cfg := config.Load()

	// Conectar a MariaDB
	db, err := database.ConnectDB(cfg.DBDSN)
	if err != nil {
		log.Printf("⚠️  No se pudo conectar a MariaDB: %v", err)
	} else {
		defer database.CloseDB()
	}

	// WebSocket hub
	hub := ws.NewHub()

	// Echo router
	e := echo.New()
	e.HideBanner = true

	// Custom error handler — consistent JSON error format
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

	// Global middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS only in dev mode (production is same-origin via embedded frontend)
	if cfg.DevMode {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{cfg.AllowedOrigins},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           300,
		}))
	}

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// API routes
	api := e.Group("/api")
	authHandler := &handlers.AuthHandler{DB: db, Config: cfg}
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)

	// Protected routes
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

	// WebSocket — JWT protected via group middleware
	protected.GET("/ws", func(c echo.Context) error {
		return ws.HandleWebSocket(hub, c)
	})

	// Static file serving (production mode only)
	if !cfg.DevMode {
		subFS, err := fs.Sub(embedFS, "client/dist")
		if err != nil {
			e.Logger.Fatal("Failed to get embedded FS:", err)
		}

		// Hashed assets — immutable cache (Vite hashes filenames)
		e.GET("/assets/*", echo.WrapHandler(http.FileServer(http.FS(subFS))))

		// SPA fallback — serve index.html for all non-API, non-asset routes
		// MUST be registered LAST, after all API routes
		e.GET("/*", func(c echo.Context) error {
			data, err := fs.ReadFile(subFS, "index.html")
			if err != nil {
				return c.NoContent(http.StatusNotFound)
			}
			return c.HTMLBlob(http.StatusOK, data)
		})
	}

	log.Printf("🚀 DaveChat API iniciado en http://0.0.0.0:%s", cfg.Port)
	if err := e.Start("0.0.0.0:" + cfg.Port); err != nil {
		log.Fatal("Error iniciando servidor:", err)
	}
}
