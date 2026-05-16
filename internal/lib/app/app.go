package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sso/internal/domain/models"
	"sso/internal/storage/postgres"
	"time"
)

func main() {

	var appName, dbPath string

	flag.StringVar(&appName, "app_name", "", "app name")
	flag.StringVar(&dbPath, "db_name", "", "db name")
	flag.Parse()

	if appName == "" {
		panic("app name is required")
	}
	if dbPath == "" {
		panic("db path is required")
	}

	err := createApp(appName, dbPath)
	if err != nil {
		log.Fatal(err)
	}
}

func createApp(name, dbPath string) error {
	const op = "lib.app.createApp"

	secret, err := models.GenerateAppSecret()
	if err != nil {
		return fmt.Errorf("%s: %w ", op, err)
	}
	fmt.Printf("secret: %s", secret)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, dbPath, 1, 1)
	if err != nil {
		return fmt.Errorf("%s: %w ", op, err)
	}
	defer pool.Close()

	repo := postgres.NewRepository(pool)

	appID, err := repo.SaveApp(ctx, name, secret)
	if err != nil {
		return fmt.Errorf("%s: %w ", op, err)
	}
	fmt.Println("app added successfully")
	fmt.Printf("id: %d", appID)
	fmt.Printf("name: %d", name)
	return nil
}
