package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	gormPG "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type DB struct {
	*gorm.DB
	sqlDB *sql.DB
}

func Connect(databaseURL string) (*DB, error) {
	gormDB, err := gorm.Open(gormPG.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	return &DB{DB: gormDB, sqlDB: sqlDB}, nil
}

func (db *DB) RunMigrations() error {
	driver, err := postgres.WithInstance(db.sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func (db *DB) MigrationVersion() (uint, bool, error) {
	driver, err := postgres.WithInstance(db.sqlDB, &postgres.Config{})
	if err != nil {
		return 0, false, err
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return 0, false, err
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return 0, false, err
	}

	version, dirty, err := m.Version()
	return version, dirty, err
}

func (db *DB) Close() error {
	return db.sqlDB.Close()
}
