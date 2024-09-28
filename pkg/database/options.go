package database

import (
	"time"

	"github.com/jackc/pgx/v5"
)

// Option -.
type Option func(*Postgres)

// MaxPoolSize -.
func MaxPoolSize(size int32) Option {
	return func(c *Postgres) {
		c.maxPoolSize = size
	}
}

// ConnAttempts -.
func ConnAttempts(attempts int32) Option {
	return func(c *Postgres) {
		c.connAttempts = attempts
	}
}

// ConnTimeout -.
func ConnTimeout(timeout time.Duration) Option {
	return func(c *Postgres) {
		c.connTimeout = timeout
	}
}

// Isolation Level -.
func Isolation(isolation pgx.TxIsoLevel) Option {
	return func(c *Postgres) {
		c.isolation = isolation
	}
}
