package dbsqlc

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(db *pgxpool.Pool) *Queries {
	return &Queries{db: db}
}

type Queries struct {
	db *pgxpool.Pool
}
