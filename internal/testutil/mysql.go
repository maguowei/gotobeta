package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

type MySQLContainer struct {
	Container testcontainers.Container
	DSN       string
}

func StartMySQL(ctx context.Context, t *testing.T) *MySQLContainer {
	t.Helper()

	startCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	c, err := mysql.Run(startCtx, "mysql:8",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("root"),
		mysql.WithPassword("root"),
	)
	if err != nil {
		t.Fatalf("start mysql container: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()

		if err := c.Terminate(cleanupCtx); err != nil {
			t.Logf("terminate mysql container: %v", err)
		}
	})

	host, err := c.Host(startCtx)
	if err != nil {
		t.Fatalf("get mysql host: %v", err)
	}
	port, err := c.MappedPort(startCtx, "3306")
	if err != nil {
		t.Fatalf("get mysql port: %v", err)
	}

	dsn := fmt.Sprintf("root:root@tcp(%s:%s)/testdb?parseTime=true", host, port.Port())
	return &MySQLContainer{Container: c, DSN: dsn}
}
