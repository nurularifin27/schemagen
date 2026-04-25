package entitygen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nurularifin27/schemagen/dbtype"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGenerateIntegrationSQLite(t *testing.T) {
	db := openSQLiteIntegrationDB(t)
	runIntegrationCase(t, db, integrationCase{
		driver:      "sqlite",
		table:       "billing_records",
		createTable: `CREATE TABLE billing_records (id integer primary key autoincrement, amount decimal not null, payload json, created_at datetime not null)`,
	})
}

func TestGenerateIntegrationPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SCHEMAGEN_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("SCHEMAGEN_TEST_POSTGRES_DSN is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	runIntegrationCase(t, db, integrationCase{
		driver:      "postgres",
		table:       "billing_records",
		createTable: `CREATE TABLE billing_records (id bigserial primary key, amount numeric(12,2) not null, payload jsonb, created_at timestamptz not null)`,
	})
}

func TestGenerateIntegrationMySQL(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SCHEMAGEN_TEST_MYSQL_DSN"))
	if dsn == "" {
		t.Skip("SCHEMAGEN_TEST_MYSQL_DSN is not set")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	runIntegrationCase(t, db, integrationCase{
		driver:      "mysql",
		table:       "billing_records",
		createTable: `CREATE TABLE billing_records (id bigint unsigned not null auto_increment primary key, amount decimal(12,2) not null, payload json, created_at datetime not null)`,
	})
}

func TestGenerateIntegrationMariaDB(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SCHEMAGEN_TEST_MARIADB_DSN"))
	if dsn == "" {
		t.Skip("SCHEMAGEN_TEST_MARIADB_DSN is not set")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mariadb: %v", err)
	}
	runIntegrationCase(t, db, integrationCase{
		driver:      "mariadb",
		table:       "billing_records",
		createTable: `CREATE TABLE billing_records (id bigint unsigned not null auto_increment primary key, amount decimal(12,2) not null, payload json, created_at datetime not null)`,
	})
}

type integrationCase struct {
	driver      string
	table       string
	createTable string
}

func runIntegrationCase(t *testing.T, db *gorm.DB, tc integrationCase) {
	t.Helper()

	if err := db.Exec(`DROP TABLE IF EXISTS ` + tc.table).Error; err != nil {
		t.Fatalf("drop table %s: %v", tc.table, err)
	}
	if err := db.Exec(tc.createTable).Error; err != nil {
		t.Fatalf("create table %s: %v", tc.table, err)
	}
	t.Cleanup(func() {
		_ = db.Exec(`DROP TABLE IF EXISTS ` + tc.table).Error
	})

	outDir := t.TempDir()
	_, err := Generate(db, Options{
		Driver:          tc.driver,
		OutDir:          outDir,
		Tables:          []string{tc.table},
		OnConflict:      "skip",
		DecimalStrategy: "string",
		JSONStrategy:    "rawmessage",
		TypeOverrides: []dbtype.Override{
			{
				Table:   tc.table,
				Column:  "amount",
				GoType:  "money.Amount",
				Imports: []string{"example.com/project/money"},
			},
		},
	})
	if err != nil {
		t.Fatalf("generate for %s: %v", tc.driver, err)
	}

	path := filepath.Join(outDir, "billing_record.go")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	text := string(content)
	required := integrationRequiredTokens(tc.driver)
	for _, token := range required {
		if !strings.Contains(text, token) {
			t.Fatalf("%s generated output missing %q:\n%s", tc.driver, token, text)
		}
	}
}

func integrationRequiredTokens(driver string) []string {
	base := []string{
		`"example.com/project/money"`,
		"money.Amount",
		"CreatedAt",
	}

	switch driver {
	case "mariadb":
		// MariaDB commonly exposes JSON columns as LONGTEXT during schema introspection,
		// so logical JSON metadata is not reliably available here.
		return append(base, "*string")
	default:
		return append(base, `"encoding/json"`, "*json.RawMessage")
	}
}

func openSQLiteIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "integration.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}
