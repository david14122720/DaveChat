package main

import (
	"fmt"
	"log"

	"DaveChat/internal/database"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error cargando el archivo .env")
	}

	// Usar DSN desde config o hardcode para test
	// En producción se carga desde .env con la clave DB_DSN
	fmt.Println("Intentando conectar a MariaDB...")

	db, err := database.ConnectDB("davechat:davechat_pass@tcp(localhost:3307)/davechat?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatalf("Error de conexión: %v", err)
	}
	defer database.CloseDB()

	// Probar una consulta simple
	var now string
	err = db.QueryRow("SELECT NOW()").Scan(&now)
	if err != nil {
		log.Fatalf("Error ejecutando consulta: %v", err)
	}

	fmt.Printf("✅ Conexión exitosa! Hora del servidor: %s\n", now)
}
