package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

var (
	db   *sql.DB
	once sync.Once
)

// ConnectDB inicializa el pool de conexiones a MariaDB.
func ConnectDB(dsn string) (*sql.DB, error) {
	var err error
	once.Do(func() {
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			err = fmt.Errorf("unable to open database: %v", err)
			return
		}
		db.SetMaxOpenConns(20)
		db.SetMaxIdleConns(5)
		if err = db.Ping(); err != nil {
			err = fmt.Errorf("unable to ping database: %v", err)
			return
		}
		log.Println("✅ Conectado a MariaDB")
	})
	return db, err
}

// GetDB devuelve la instancia de la base de datos.
func GetDB() *sql.DB {
	return db
}

// CloseDB cierra la conexión a la base de datos.
func CloseDB() {
	if db != nil {
		db.Close()
	}
}
