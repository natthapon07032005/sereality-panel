package database

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteBackupSuffix = ".backup"

// CreateBackup creates a validated, WAL-aware SQLite snapshot at destinationPath atomically.
func CreateBackup(sourcePath, destinationPath string) error {
	if valid, err := IsSQLiteFile(sourcePath); err != nil {
		return err
	} else if !valid {
		return fmt.Errorf("source is not a SQLite database")
	}
	return vacuumSQLiteSnapshot(sourcePath, destinationPath)
}

// RestoreDatabase restores a validated SQLite file and keeps the active DB
// untouched when validation or activation fails.
func RestoreDatabase(currentPath, candidatePath string) error {
	candidate, err := os.Open(candidatePath)
	if err != nil {
		return err
	}
	defer candidate.Close()
	_, err = RestoreSQLite(currentPath, candidate)
	return err
}

func IsSQLiteFile(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return IsSQLiteDB(file)
}

func ValidateSQLiteDB(filePath string) error {
	valid, err := IsSQLiteFile(filePath)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("file is not a SQLite database")
	}
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return err
	}
	defer db.Close()
	var result string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("SQLite integrity check failed: %s", result)
	}
	return nil
}

func BackupSQLite(destinationPath string) error {
	if db == nil {
		return fmt.Errorf("database is not initialized")
	}
	if err := Checkpoint(); err != nil {
		return fmt.Errorf("checkpoint database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	rows, err := sqlDB.Query("PRAGMA database_list")
	if err != nil {
		return err
	}
	defer rows.Close()
	var sourcePath string
	for rows.Next() {
		var sequence int
		var name, file string
		if err := rows.Scan(&sequence, &name, &file); err != nil {
			return err
		}
		if name == "main" {
			sourcePath = file
			break
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if sourcePath == "" {
		return fmt.Errorf("active SQLite path is unavailable")
	}
	return copySQLiteFile(sourcePath, destinationPath)
}

func RestoreSQLite(currentPath string, candidate io.Reader) (string, error) {
	backupPath := currentPath + sqliteBackupSuffix
	if _, err := os.Stat(currentPath); err == nil {
		if db != nil {
			if err := Checkpoint(); err != nil {
				return "", fmt.Errorf("checkpoint current database: %w", err)
			}
		}
		if err := CreateBackup(currentPath, backupPath); err != nil {
			return "", fmt.Errorf("backup current database: %w", err)
		}
	}
	candidateFile, err := os.CreateTemp(filepath.Dir(currentPath), ".sereality-candidate-*.db")
	if err != nil {
		return backupPath, err
	}
	candidatePath := candidateFile.Name()
	defer os.Remove(candidatePath)
	if err := candidateFile.Chmod(0600); err != nil {
		candidateFile.Close()
		return backupPath, err
	}
	if _, err := io.Copy(candidateFile, candidate); err != nil {
		candidateFile.Close()
		return backupPath, err
	}
	if err := candidateFile.Sync(); err != nil {
		candidateFile.Close()
		return backupPath, err
	}
	if err := candidateFile.Close(); err != nil {
		return backupPath, err
	}
	if err := ValidateSQLiteDB(candidatePath); err != nil {
		return backupPath, fmt.Errorf("validate restore candidate: %w", err)
	}
	if err := validateSchemaForActivation(candidatePath); err != nil {
		return backupPath, err
	}
	if err := CloseDB(); err != nil {
		return backupPath, fmt.Errorf("close active database: %w", err)
	}
	if err := activateRestoreCandidate(candidatePath, currentPath); err != nil {
		_ = copySQLiteFile(backupPath, currentPath)
		_ = InitDB(currentPath)
		return backupPath, fmt.Errorf("activate restore candidate: %w", err)
	}
	if err := InitDB(currentPath); err != nil {
		_ = copySQLiteFile(backupPath, currentPath)
		_ = InitDB(currentPath)
		return backupPath, fmt.Errorf("reopen restored database: %w", err)
	}
	return backupPath, nil
}

func activateRestoreCandidate(candidatePath, currentPath string) error {
	return replaceFileAtomically(candidatePath, currentPath)
}

func validateSchemaForActivation(filePath string) error {
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return err
	}
	defer db.Close()
	var objectType string
	if err := db.QueryRow("SELECT type FROM sqlite_master WHERE name = 'users'").Scan(&objectType); err != nil {
		return fmt.Errorf("validate users table: %w", err)
	}
	if objectType != "table" {
		return fmt.Errorf("validate users table: expected table, got %s", objectType)
	}
	return nil
}

func copySQLiteFile(sourcePath, destinationPath string) error {
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(destinationPath), ".sereality-copy-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		temp.Close()
		return err
	}
	_, copyErr := io.Copy(temp, source)
	source.Close()
	if copyErr != nil {
		temp.Close()
		return copyErr
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return replaceFileAtomically(tempPath, destinationPath)
}

// vacuumSQLiteSnapshot asks SQLite to materialize a consistent snapshot. This
// includes committed records that are still in a WAL sidecar, unlike copying
// the main database file directly.
func vacuumSQLiteSnapshot(sourcePath, destinationPath string) error {
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0700); err != nil {
		return err
	}
	source, err := sql.Open("sqlite3", sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	source.SetMaxOpenConns(1)

	temp, err := os.CreateTemp(filepath.Dir(destinationPath), ".sereality-snapshot-*.db")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	defer os.Remove(tempPath)
	if err := os.Remove(tempPath); err != nil {
		return err
	}

	quotedPath := strings.ReplaceAll(tempPath, "'", "''")
	if _, err := source.Exec("VACUUM INTO '" + quotedPath + "'"); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0600); err != nil {
		return err
	}
	if err := ValidateSQLiteDB(tempPath); err != nil {
		return fmt.Errorf("validate SQLite snapshot: %w", err)
	}
	return replaceFileAtomically(tempPath, destinationPath)
}

func replaceFileAtomically(sourcePath, destinationPath string) error {
	if err := os.Rename(sourcePath, destinationPath); err == nil {
		return nil
	}
	return replaceFileAtomicallyPlatform(sourcePath, destinationPath)
}
