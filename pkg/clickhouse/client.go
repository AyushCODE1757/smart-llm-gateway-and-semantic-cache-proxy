package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

type Client struct {
	conn ch.Conn
}

func NewClient() (*Client, error) {
	host := os.Getenv("CLICKHOUSE_HOST")
	port := os.Getenv("CLICKHOUSE_PORT")
	user := os.Getenv("CLICKHOUSE_USER")
	password := os.Getenv("CLICKHOUSE_PASSWORD")
	database := os.Getenv("CLICKHOUSE_DATABASE")
	if host == "" || port == "" || user == "" || password == "" || database == "" {
		return nil, fmt.Errorf("missing ClickHouse environment variables")
	}
	conn, err := ch.Open(&ch.Options{
		Addr:     []string{fmt.Sprintf("%s:%s", host, port)},
		Protocol: ch.HTTP,
		Auth: ch.Auth{
			Database: database,
			Username: user,
			Password: password,
		},
		TLS:         &tls.Config{},
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}
	return &Client{conn: conn}, nil
}
func (c *Client) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.conn.Ping(ctx)
}
