package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func requireDecimalEqual(t *testing.T, expected, actual decimal.Decimal) {
	t.Helper()
	require.True(t, expected.Equal(actual), "expected %s, got %s", expected, actual)
}

const (
	dbDriver = "postgres"
	// TODO: Use an environment variable
	dbSource = "postgresql://root:secret@localhost:5432/basic_bank_app?sslmode=disable"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run())
}
