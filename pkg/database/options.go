package database

// Option -.
type Option func(*Postgres)

// MaxPoolSize -.
func MaxPoolSize(size int32) Option {
	return func(c *Postgres) {
		c.maxPoolSize = size
	}
}

// MinPoolSize -.
func MinPoolSize(size int32) Option {
	return func(c *Postgres) {
		c.minPoolSize = size
	}
}

// ConnAttempts -.
func ConnAttempts(attempts int32) Option {
	return func(c *Postgres) {
		c.connAttempts = attempts
	}
}

// ConnTimeout -.
func ConnTimeout(timeout int) Option {
	return func(c *Postgres) {
		c.connTimeout = timeout
	}
}

// HealthCheckPeriod -.
func HealthCheckPeriod(period int) Option {
	return func(c *Postgres) {
		c.healthCheckPeriod = period
	}
}
