package database

import (
	"bytes"
	"io"
	"io/fs"
	"log"
	"os"
	"path"

	"x-ui/config"
	"x-ui/database/model"
	"x-ui/xray"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

const (
	defaultUsername = "admin"
	defaultPassword = "admin"
	defaultSecret   = ""
)

func initModels() error {
	models := []interface{}{
		&model.User{},
		&model.Inbound{},
		&model.OutboundTraffics{},
		&model.Setting{},
		&model.InboundClientIps{},
		&model.Package{},
		&model.Node{},
		&model.Subscription{},
		&xray.ClientTraffic{},
	}
	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			log.Printf("Error auto migrating model: %v", err)
			return err
		}
	}
	return nil
}

func initUser() error {
	empty, err := isTableEmpty("users")
	if err != nil {
		log.Printf("Error checking if users table is empty: %v", err)
		return err
	}
	if empty {
		hashedPassword, err := defaultPasswordHash()
		if err != nil {
			return err
		}
		user := &model.User{
			Username:    defaultUsername,
			Password:    hashedPassword,
			LoginSecret: defaultSecret,
		}
		return db.Create(user).Error
	}
	return nil
}

func defaultPasswordHash() (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	return string(hash), err
}

func isTableEmpty(tableName string) (bool, error) {
	var count int64
	err := db.Table(tableName).Count(&count).Error
	return count == 0, err
}

func InitDB(dbPath string) error {
	dir := path.Dir(dbPath)
	err := os.MkdirAll(dir, fs.ModePerm)
	if err != nil {
		return err
	}

	var gormLogger logger.Interface

	if config.IsDebug() {
		gormLogger = logger.Default
	} else {
		gormLogger = logger.Discard
	}

	c := &gorm.Config{
		Logger: gormLogger,
	}
	db, err = gorm.Open(sqlite.Open(dbPath), c)
	if err != nil {
		return err
	}
	if sqlDB, err := db.DB(); err != nil {
		return err
	} else {
		// Keep SQLite on one connection so the connection-scoped FK pragma is
		// effective for every management operation and restore checkpoint.
		sqlDB.SetMaxOpenConns(1)
		if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
			_ = sqlDB.Close()
			return err
		}
	}

	if err := initModels(); err != nil {
		_ = CloseDB()
		return err
	}
	if err := initUser(); err != nil {
		_ = CloseDB()
		return err
	}

	return nil
}

func CloseDB() error {
	if db != nil {
		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		err = sqlDB.Close()
		// Keep the closed handle during a restore transition. Callers that race
		// with the transition receive a database-closed error instead of
		// dereferencing a nil shared pointer; InitDB replaces it when ready.
		return err
	}
	return nil
}

func GetDB() *gorm.DB {
	return db
}

func IsNotFound(err error) bool {
	return err == gorm.ErrRecordNotFound
}

func IsSQLiteDB(file io.ReaderAt) (bool, error) {
	signature := []byte("SQLite format 3\x00")
	buf := make([]byte, len(signature))
	_, err := file.ReadAt(buf, 0)
	if err != nil {
		return false, err
	}
	return bytes.Equal(buf, signature), nil
}

func Checkpoint() error {
	// Update WAL
	err := db.Exec("PRAGMA wal_checkpoint;").Error
	if err != nil {
		return err
	}
	return nil
}
