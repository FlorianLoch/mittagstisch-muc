package database

import (
	"context"
	"database/sql"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"errors"
	"fmt"
	"github.com/florianloch/mittagstisch/ent"
	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
)

var ErrNotFound = errors.New("not found")

type DB struct {
	client *ent.Client
}

func New(ctx context.Context, dsn string) (*DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database connection: %w", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, db)

	client := ent.NewClient(ent.Driver(drv))

	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("creating schema resources: %w", err)
	}

	return &DB{client: client}, nil
}
