package database

import "testing"

func TestConfigDSN(t *testing.T) {
	config := Config{
		Host:     "db.internal",
		Port:     "5432",
		Name:     "billing",
		User:     "billing-user",
		Password: "p@ss word",
		Sslmode:  "require",
	}
	want := "postgres://billing-user:p%40ss%20word@db.internal:5432/billing?sslmode=require"
	if got := config.DSN(); got != want {
		t.Fatalf("DSN() = %q, want %q", got, want)
	}
}
