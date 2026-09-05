package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"x-ui/database"
)

const (
	DefaultAutomaticBackupRetention     = 7
	MinAutomaticBackupRetention         = 1
	MaxAutomaticBackupRetention         = 30
	DefaultAutomaticBackupIntervalHours = 24
	MinAutomaticBackupIntervalHours     = 1
	MaxAutomaticBackupIntervalHours     = 168
	DefaultAutomaticBackupDirectory     = "backups"
	automaticBackupPrefix               = "sereality-"
)

// AutomaticBackupConfig controls periodic local database snapshots.
type AutomaticBackupConfig struct {
	Enabled       bool   `json:"enabled"`
	IntervalHours int    `json:"intervalHours"`
	Retention     int    `json:"retention"`
	Directory     string `json:"directory"`
}

func NormalizeAutomaticBackupConfig(input AutomaticBackupConfig) AutomaticBackupConfig {
	if input.IntervalHours < MinAutomaticBackupIntervalHours {
		input.IntervalHours = DefaultAutomaticBackupIntervalHours
	}
	if input.IntervalHours > MaxAutomaticBackupIntervalHours {
		input.IntervalHours = MaxAutomaticBackupIntervalHours
	}
	if input.Retention < MinAutomaticBackupRetention {
		input.Retention = DefaultAutomaticBackupRetention
	}
	if input.Retention > MaxAutomaticBackupRetention {
		input.Retention = MaxAutomaticBackupRetention
	}
	input.Directory = strings.TrimSpace(input.Directory)
	if input.Directory == "" {
		input.Directory = DefaultAutomaticBackupDirectory
	}
	return input
}

func ShouldRunAutomaticBackup(lastRun, now time.Time, interval time.Duration) bool {
	if interval <= 0 || lastRun.IsZero() {
		return true
	}
	if lastRun.After(now) {
		return true
	}
	return !now.Before(lastRun.Add(interval))
}

// AutomaticBackupFileName returns a stable, sortable filename for one snapshot.
func AutomaticBackupFileName(now time.Time) string {
	return fmt.Sprintf("%s%s.db", automaticBackupPrefix, now.UTC().Format("20060102-150405"))
}

// CreateAutomaticBackup writes one validated SQLite snapshot and then enforces retention.
func CreateAutomaticBackup(sourcePath, directory string, now time.Time, retention int) (string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return "", fmt.Errorf("backup source path is required")
	}
	if strings.TrimSpace(directory) == "" {
		return "", fmt.Errorf("backup directory is required")
	}
	if retention < MinAutomaticBackupRetention || retention > MaxAutomaticBackupRetention {
		return "", fmt.Errorf("backup retention is outside the allowed range")
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return "", err
	}
	destination := filepath.Join(directory, AutomaticBackupFileName(now))
	if err := database.CreateBackup(sourcePath, destination); err != nil {
		return "", err
	}
	if _, err := PruneAutomaticBackups(directory, retention); err != nil {
		return "", err
	}
	return destination, nil
}

// ValidateBackupFile verifies that a path is a readable SQLite snapshot.
func ValidateBackupFile(path string) error {
	if strings.TrimSpace(path) == "" || !strings.HasPrefix(filepath.Base(path), automaticBackupPrefix) || !strings.HasSuffix(path, ".db") {
		return fmt.Errorf("invalid automatic backup path")
	}
	return database.ValidateSQLiteDB(path)
}

// PruneAutomaticBackups only removes files created by this feature.
func PruneAutomaticBackups(directory string, retention int) (int, error) {
	if retention < MinAutomaticBackupRetention || retention > MaxAutomaticBackupRetention {
		return 0, fmt.Errorf("backup retention is outside the allowed range")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, err
	}
	type backupEntry struct {
		path    string
		modTime time.Time
	}
	backups := make([]backupEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), automaticBackupPrefix) || !strings.HasSuffix(entry.Name(), ".db") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return 0, err
		}
		backups = append(backups, backupEntry{path: filepath.Join(directory, entry.Name()), modTime: info.ModTime()})
	}
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].modTime.Equal(backups[j].modTime) {
			return backups[i].path > backups[j].path
		}
		return backups[i].modTime.After(backups[j].modTime)
	})
	removed := 0
	if len(backups) <= retention {
		return 0, nil
	}
	for _, backup := range backups[retention:] {
		if err := os.Remove(backup.path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}
