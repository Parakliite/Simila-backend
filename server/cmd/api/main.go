package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var version = "1.0.0"
const port = 8080

type dbInfo struct {
	dsn string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime string
}

type apiConfig struct {
	port int
	db dbInfo
	environment string
}

type application struct {
	config *apiConfig
	logger *log.Logger
	models data.Models
}

func main() {
	godotenv.Load()

	logger := log.New(os.Stdout, "", log.Ldate | log.Ltime)

	var cfg apiConfig
	cfg.port = port
	cfg.db.dsn = os.Getenv("DB_URL")
	cfg.db.maxOpenConns = 25
    cfg.db.maxIdleConns = 10
    cfg.db.maxIdleTime = "15m"

	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	logger.Printf("database connection pool established")

	app := &application{
		config: &cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	mux := app.routes()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		IdleTimeout: time.Minute,
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Serving on port %s\n", port)
	log.Fatal(server.ListenAndServe())
}

func openDB(cfg apiConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}