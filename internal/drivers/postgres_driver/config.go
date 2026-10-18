package postgres_driver

import (
	"net"
	"net/url"
)

// Config holds the parameters needed to connect to a Postgres database.
type Config struct {
	host     string
	port     string
	username string
	password string
	database string
	options  map[string]string
}

// NewConfig creates a Config. Options are added to the connection string as query parameters
// (e.g. sslmode, application_name, pool_max_conns).
func NewConfig(host, port, username, password, database string, options map[string]string) Config {
	return Config{
		host:     host,
		port:     port,
		username: username,
		password: password,
		database: database,
		options:  options,
	}
}

// GetConnectionString returns the config as a postgres:// URL, escaping every part.
func (c Config) GetConnectionString() string {
	query := url.Values{}
	for key, value := range c.options {
		query.Set(key, value)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.username, c.password),
		Host:     net.JoinHostPort(c.host, c.port),
		Path:     "/" + c.database,
		RawQuery: query.Encode(),
	}
	return u.String()
}
