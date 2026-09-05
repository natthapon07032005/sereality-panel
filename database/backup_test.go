package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"x-ui/database/model"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateBackupCreatesConsistentSnapshotFromWAL(t *testing.T) {
	dbPath := initBackupTestDatabase(t)
	sqlDB := activeSQLDB(t)
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("enable WAL mode: %v", err)
	}
	if _, err := sqlDB.Exec("PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatalf("disable automatic checkpointing: %v", err)
	}
	if err := GetDB().Create(&model.Setting{Key: "backup-marker", Value: "committed-in-wal"}).Error; err != nil {
		t.Fatalf("seed WAL record: %v", err)
	}
	if _, err := os.Stat(dbPath + "-wal"); err != nil {
		t.Fatalf("expected WAL sidecar after committed write: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "snapshot.db")
	if err := CreateBackup(dbPath, backupPath); err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if got := settingValue(t, backupPath, "backup-marker"); got != "committed-in-wal" {
		t.Fatalf("backup marker = %q, want committed WAL value", got)
	}
}

func TestRestoreDatabaseRejectsHeaderOnlyInputBeforeCreatingTarget(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "active.db")
	headerOnlyPath := filepath.Join(t.TempDir(), "header-only.db")
	headerOnlySQLite := append([]byte("SQLite format 3\x00"), make([]byte, 4096)...)
	if err := os.WriteFile(headerOnlyPath, headerOnlySQLite, 0600); err != nil {
		t.Fatalf("write header-only input: %v", err)
	}
	if err := RestoreDatabase(targetPath, headerOnlyPath); err == nil {
		t.Fatal("RestoreDatabase accepted a header-only non-database")
	}
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("restore created a target from rejected input: %v", err)
	}
}

func TestActivateRestoreCandidateReplacesExistingDatabaseFile(t *testing.T) {
	directory := t.TempDir()
	currentPath := filepath.Join(directory, "active.db")
	candidatePath := filepath.Join(directory, "candidate.db")
	if err := os.WriteFile(currentPath, []byte("old"), 0600); err != nil {
		t.Fatalf("write current database: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte("new"), 0600); err != nil {
		t.Fatalf("write restore candidate: %v", err)
	}

	if err := activateRestoreCandidate(candidatePath, currentPath); err != nil {
		t.Fatalf("activateRestoreCandidate: %v", err)
	}
	contents, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("read activated database: %v", err)
	}
	if string(contents) != "new" {
		t.Fatalf("activated database = %q, want %q", contents, "new")
	}
}

func TestRestoreDatabaseRestoresOriginalWhenActivationFails(t *testing.T) {
	dbPath := initBackupTestDatabase(t)
	if err := GetDB().Create(&model.Setting{Key: "rollback-marker", Value: "original-value"}).Error; err != nil {
		t.Fatalf("seed active database: %v", err)
	}

	incompatiblePath := filepath.Join(t.TempDir(), "incompatible.db")
	createActivationFailingSQLite(t, incompatiblePath)
	incompatibleFile, err := os.Open(incompatiblePath)
	if err != nil {
		t.Fatalf("open incompatible database: %v", err)
	}
	defer incompatibleFile.Close()

	backupPath, err := RestoreSQLite(dbPath, incompatibleFile)
	if err == nil {
		t.Fatal("RestoreSQLite accepted a database that cannot be activated")
	}
	if backupPath == "" {
		t.Fatalf("RestoreSQLite did not retain a pre-restore backup: %v", err)
	}
	if got := settingValue(t, backupPath, "rollback-marker"); got != "original-value" {
		t.Fatalf("pre-restore backup marker = %q, want original value", got)
	}
	if got := activeSettingValue(t, "rollback-marker"); got != "original-value" {
		t.Fatalf("active marker after failed activation = %q, want original value", got)
	}
}

func TestRestoreDatabaseActivatesValidCandidate(t *testing.T) {
	dbPath := initBackupTestDatabase(t)
	if err := GetDB().Create(&model.Setting{Key: "original-marker", Value: "original-value"}).Error; err != nil {
		t.Fatalf("seed active database: %v", err)
	}

	candidatePath := filepath.Join(t.TempDir(), "candidate.db")
	createCompatibleSQLite(t, candidatePath, "candidate-marker", "candidate-value")
	if err := RestoreDatabase(dbPath, candidatePath); err != nil {
		t.Fatalf("RestoreDatabase: %v", err)
	}
	if got := activeSettingValue(t, "candidate-marker"); got != "candidate-value" {
		t.Fatalf("active marker after restore = %q, want candidate value", got)
	}
}

func initBackupTestDatabase(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "x-ui.db")
	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() {
		if err := CloseDB(); err != nil {
			t.Errorf("CloseDB: %v", err)
		}
	})
	return dbPath
}

func createActivationFailingSQLite(t *testing.T, dbPath string) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open incompatible sqlite database: %v", err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, password TEXT, login_secret TEXT)"); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := sqlDB.Exec("CREATE VIEW inbounds AS SELECT 1 AS id"); err != nil {
		t.Fatalf("create incompatible inbounds view: %v", err)
	}
}

func createCompatibleSQLite(t *testing.T, dbPath, key, value string) {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open compatible sqlite database: %v", err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, password TEXT, login_secret TEXT)"); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := sqlDB.Exec("CREATE TABLE settings (id INTEGER PRIMARY KEY, key TEXT, value TEXT)"); err != nil {
		t.Fatalf("create settings table: %v", err)
	}
	if _, err := sqlDB.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", key, value); err != nil {
		t.Fatalf("seed compatible sqlite database: %v", err)
	}
}

func activeSQLDB(t *testing.T) *sql.DB {
	t.Helper()
	if GetDB() == nil {
		t.Fatal("active database is nil")
	}
	sqlDB, err := GetDB().DB()
	if err != nil {
		t.Fatalf("get active sql database: %v", err)
	}
	return sqlDB
}

func activeSettingValue(t *testing.T, key string) string {
	t.Helper()
	return settingValueFromSQLDB(t, activeSQLDB(t), key)
}

func settingValue(t *testing.T, dbPath, key string) string {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite database %q: %v", dbPath, err)
	}
	defer sqlDB.Close()
	return settingValueFromSQLDB(t, sqlDB, key)
}

func settingValueFromSQLDB(t *testing.T, sqlDB *sql.DB, key string) string {
	t.Helper()
	var value string
	if err := sqlDB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value); err != nil {
		t.Fatalf("read setting %q: %v", key, err)
	}
	return value
}
