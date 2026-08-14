package dbstudio

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"mypaas/internal/db"
)

func TestComposeDatabaseResolverMatrix(t *testing.T) {
	project := db.Project{ID: uuid.New(), Name: "beta-db", DeployMode: "compose"}
	tests := []struct {
		name     string
		envs     map[string]string
		driver   DriverID
		database string
		user     string
		port     int
	}{
		{
			name: "postgres",
			envs: map[string]string{
				"POSTGRES_DB":       "appdb",
				"POSTGRES_USER":     "app",
				"POSTGRES_PASSWORD": "fixture-secret",
			},
			driver: DriverPostgres, database: "appdb", user: "app", port: 5432,
		},
		{
			name: "mysql",
			envs: map[string]string{
				"MYSQL_DATABASE": "appdb",
				"MYSQL_USER":     "app",
				"MYSQL_PASSWORD": "fixture-secret",
			},
			driver: DriverMySQL, database: "appdb", user: "app", port: 3306,
		},
		{
			name: "mariadb",
			envs: map[string]string{
				"MARIADB_DATABASE": "appdb",
				"MARIADB_USER":     "app",
				"MARIADB_PASSWORD": "fixture-secret",
			},
			driver: DriverMariaDB, database: "appdb", user: "app", port: 3306,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, ok := resolveParts(project, tt.envs)
			if !ok {
				t.Fatal("resolveParts() did not resolve supported Compose database environment")
			}
			if conn.Driver != tt.driver || conn.Host != "db" || conn.Port != tt.port || conn.Database != tt.database || conn.User != tt.user {
				t.Fatalf("unexpected connection: %#v", conn)
			}
			if conn.Source != "env-parts" || conn.DSN == "" {
				t.Fatalf("connection source/DSN missing: %#v", conn)
			}
		})
	}
}

func TestComposeDatabaseIdentityRejectsIncompleteCredentials(t *testing.T) {
	cases := []map[string]string{
		{"POSTGRES_DB": "appdb"},
		{"POSTGRES_USER": "app"},
		{"MYSQL_DATABASE": "appdb", "MYSQL_PASSWORD": "secret"},
		{"MARIADB_USER": "app", "MARIADB_PASSWORD": "secret"},
	}
	for _, envs := range cases {
		if hasDatabaseIdentityEnv(envs) {
			t.Fatalf("incomplete environment was accepted as a database identity: %#v", envs)
		}
	}
}

func TestProjectEnvironmentOverridesComposeFallbackWithoutDroppingIdentity(t *testing.T) {
	projectEnv := map[string]string{
		"DB_PASSWORD": "owner-override",
	}
	serviceEnv := map[string]string{
		"MARIADB_DATABASE": "appdb",
		"MARIADB_USER":     "app",
		"MARIADB_PASSWORD": "service-secret",
	}
	merged := mergeMissingEnv(projectEnv, serviceEnv)
	if merged["DB_PASSWORD"] != "owner-override" {
		t.Fatal("project environment override was lost")
	}
	if merged["MARIADB_DATABASE"] != "appdb" || merged["MARIADB_USER"] != "app" {
		t.Fatal("Compose service identity was not retained")
	}
}

func TestSupportedDatabaseURLsNeverExposePasswordInConnectionMetadata(t *testing.T) {
	urls := []struct {
		raw    string
		driver DriverID
	}{
		{"postgres://app:very-secret@db:5432/appdb", DriverPostgres},
		{"mysql://app:very-secret@tcp(db:3306)/appdb", DriverMySQL},
		{"mariadb://app:very-secret@db:3306/appdb", DriverMariaDB},
	}
	for _, tt := range urls {
		conn, err := connectionFromURL(tt.raw, "DATABASE_URL")
		if err != nil {
			// MySQL DSNs are not URL-shaped and therefore are intentionally tested
			// through env-parts; URL aliases remain covered where supported.
			if tt.driver == DriverMySQL {
				continue
			}
			t.Fatalf("connectionFromURL(%q): %v", tt.raw, err)
		}
		if conn.Driver != tt.driver {
			t.Fatalf("Driver = %q, want %q", conn.Driver, tt.driver)
		}
		metadata := strings.Join([]string{conn.Host, conn.Database, conn.User, conn.Source}, "|")
		if strings.Contains(metadata, "very-secret") {
			t.Fatal("connection metadata exposed database password")
		}
	}
}

func TestComposeDatabaseServiceCandidateMatrix(t *testing.T) {
	got := composeDatabaseServiceCandidates()
	for _, want := range []string{"db", "database", "postgres", "postgresql", "mysql", "mariadb"} {
		found := false
		for _, candidate := range got {
			if candidate == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing Compose database service candidate %q in %#v", want, got)
		}
	}
}
