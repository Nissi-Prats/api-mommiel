package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var (
	db   *sql.DB
	once sync.Once
)

func GetDB() *sql.DB {
	once.Do(func() {
		_ = godotenv.Load()

		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		dbname := os.Getenv("DB_NAME")

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)

		var err error
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Fatalf("Error configurando la BD: %v", err)
		}

		if err = db.Ping(); err != nil {
			log.Fatalf("No se pudo conectar a la base de datos: %v", err)
		}
		fmt.Println("¡Conexión Exitosa a la Base de Datos mediante Singleton!")
	})
	return db
}