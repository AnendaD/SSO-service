package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var databaseURL, migrationsPath, migrationsTable string

	flag.StringVar(&databaseURL, "database-url", "", "database URL (postgres://...)")
	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&migrationsTable, "migrations-table", "migrations", "name of migrations table")
	flag.Parse()

	if databaseURL == "" {
		// Пробуем из переменной окружения
		databaseURL = os.Getenv("DATABASE_URL")
	}

	if databaseURL == "" {
		log.Fatal("database-url is required (or set DATABASE_URL env)")
	}
	if migrationsPath == "" {
		log.Fatal("migrations-path is required")
	}

	log.Printf("Connecting to database...")
	log.Printf("Migrations path: %s", migrationsPath)

	// Правильно формируем URL с параметром таблицы миграций
	var fullURL string
	if strings.Contains(databaseURL, "?") {
		// Если в URL уже есть параметры, добавляем через &
		fullURL = fmt.Sprintf("%s&x-migrations-table=%s", databaseURL, migrationsTable)
	} else {
		// Если параметров нет, добавляем через ?
		fullURL = fmt.Sprintf("%s?x-migrations-table=%s", databaseURL, migrationsTable)
	}

	m, err := migrate.New(
		"file://"+migrationsPath,
		fullURL,
	)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("no migrations to apply")
			return
		}
		log.Fatalf("failed to apply migrations: %v", err)
	}

	log.Println("migrations applied successfully")
}
