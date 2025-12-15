package data

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" 
)

type Postgres struct {
	DB *sql.DB
}

func NewPostgres() (*Postgres, error) {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB_NAME")

	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "require" // default require for production (RDS)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, pass, host, port, dbname, sslmode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(50)               // max pool
	db.SetMaxIdleConns(5)                // min idle
	db.SetConnMaxIdleTime(60 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	log.Println("PostgreSQL connected")

	return &Postgres{DB: db}, nil
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}
