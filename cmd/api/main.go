package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"greenlight.ducndh.net/internal/data"
	"greenlight.ducndh.net/internal/jsonlog"
	"greenlight.ducndh.net/internal/mailer"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleTime  string
	}

	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {

	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.ducndh.net>", "SMTP sender")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	logger.PrintInfo("database connection pool established", nil)

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	// Timeout hanging connecting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Creating the config for the pgxpool
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}

	config, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	config.MinConns = 5

	config.MaxConns = int32(cfg.db.maxOpenConns)
	config.MaxConnIdleTime = duration

	// Create the pool and ping the context
	dbpool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		return nil, err
	}
	err = dbpool.Ping(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to ping the database: %v\n", err)
		return nil, err
	}

	return dbpool, nil
}
