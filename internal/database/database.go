package database

import (
	"fmt"
	"log"
	"time"

	"construct/oracle/internal/config"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DBOracle   = "oracle"
	DBAccounts = "accounts"
)

var databases = map[string]*gorm.DB{}

func Get(name string) *gorm.DB {
	db, ok := databases[name]
	if !ok {
		log.Fatalf("unknown database %q", name)
	}
	return db
}

func Init(cfg *config.Config) {
	databases[DBOracle] = openDB(cfg, cfg.DBNameOracle)
	databases[DBAccounts] = openDB(cfg, cfg.DBNameAccounts)
}

func openDB(cfg *config.Config, dbName string) *gorm.DB {
	var dialector gorm.Dialector
	switch cfg.DBDriver {
	case "postgres":
		sslmode := "disable"
		if cfg.DBSSL == "true" {
			sslmode = "require"
		}
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, dbName, sslmode)
		dialector = postgres.Open(dsn)
	default:
		tls := "false"
		if cfg.DBSSL == "true" {
			tls = "true"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
			cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, dbName, tls)
		dialector = mysql.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", dbName, err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(50)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}
	return db
}
