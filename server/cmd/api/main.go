package main

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/jsonlog"
	"github.com/deltron-fr/filmbox/server/internal/mailer"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var version = "1.0.0"

const port = 8080

type dbInfo struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type apiConfig struct {
	port        int
	db          dbInfo
	environment string
	smtp        struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	tmdbToken   string
	tmdbBaseURL string
}

type application struct {
	config *apiConfig
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	godotenv.Load()

	var cfg apiConfig
	cfg.port = port
	cfg.environment = "development"
	cfg.db.dsn = os.Getenv("DB_URL")
	cfg.db.maxOpenConns = 25
	cfg.db.maxIdleConns = 10
	cfg.db.maxIdleTime = "15m"

	cfg.tmdbToken = os.Getenv("TMDB_TOKEN")
	cfg.tmdbBaseURL = "https://api.themoviedb.org"

	cfg.smtp.host = os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	cfg.smtp.port, _ = strconv.Atoi(smtpPort)
	cfg.smtp.username = os.Getenv("SMTP_USERNAME")
	cfg.smtp.password = os.Getenv("SMTP_PASSWORD")
	cfg.smtp.sender = "Cinefilm <co-reply@cinefilm.net>"

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	app := &application{
		config: &cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(
			cfg.smtp.host,
			cfg.smtp.port,
			cfg.smtp.username,
			cfg.smtp.password,
			cfg.smtp.sender,
		),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
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

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
