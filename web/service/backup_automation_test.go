package service_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"x-ui/web/service"
)

func TestAutomaticBackupConfigNormalizesSafeDefaultsAndBounds(t *testing.T) {
	config := service.NormalizeAutomaticBackupConfig(service.AutomaticBackupConfig{
		Enabled:       true,
		IntervalHours: 999,
		Retention:     0,
		Directory:     "  ",
	})

	if !config.Enabled || config.IntervalHours != service.MaxAutomaticBackupIntervalHours || config.Retention != service.DefaultAutomaticBackupRetention {
		t.Fatalf("normalized config = %+v", config)
	}
	if config.Directory != service.DefaultAutomaticBackupDirectory {
		t.Fatalf("directory = %q, want %q", config.Directory, service.DefaultAutomaticBackupDirectory)
	}
}

func TestShouldRunAutomaticBackupHonorsInterval(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if !service.ShouldRunAutomaticBackup(time.Time{}, now, 24*time.Hour) {
		t.Fatal("backup did not run when no previous backup exists")
	}
	if service.ShouldRunAutomaticBackup(now.Add(-23*time.Hour), now, 24*time.Hour) {
		t.Fatal("backup ran before interval elapsed")
	}
	if !service.ShouldRunAutomaticBackup(now.Add(time.Hour), now, 24*time.Hour) {
		t.Fatal("backup did not run when previous timestamp was in the future")
	}
}

func TestPruneAutomaticBackupsKeepsNewestMatchingFilesOnly(t *testing.T) {
	dir := t.TempDir()
	old := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i, name := range []string{
		"sereality-20260901-120000.db",
		"sereality-20260902-120000.db",
		"sereality-20260903-120000.db",
		"sereality-20260904-120000.db",
	} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("backup"), 0600); err != nil {
			t.Fatal(err)
		}
		stamp := old.Add(time.Duration(i) * 24 * time.Hour)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := service.PruneAutomaticBackups(dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	for _, name := range []string{"sereality-20260903-120000.db", "sereality-20260904-120000.db", "notes.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s was not kept: %v", name, err)
		}
	}
	for _, name := range []string{"sereality-20260901-120000.db", "sereality-20260902-120000.db"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s was not pruned", name)
		}
	}
}

func TestCreateAutomaticBackupProducesValidatedSnapshot(t *testing.T) {
	source := filepath.Join(t.TempDir(), "panel.db")
	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT); INSERT INTO settings(key, value) VALUES ('theme', 'pink');"); err != nil {
		db.Close()
		if strings.Contains(err.Error(), "CGO_ENABLED=0") {
			t.Skip("SQLite integration requires CGO")
		}
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	directory := filepath.Join(t.TempDir(), "backups")
	path, err := service.CreateAutomaticBackup(source, directory, time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC), 2)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "sereality-20260905-120000.db" {
		t.Fatalf("backup path = %q", path)
	}
	if err := service.ValidateBackupFile(path); err != nil {
		t.Fatalf("ValidateBackupFile() error = %v", err)
	}
}
