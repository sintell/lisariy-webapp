package server

import (
	"fmt"

	"github.com/jinzhu/gorm"
	// using form dialect here
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

var (
	store *Store
)

type Store struct {
	db *gorm.DB
}

func NewStore(cfg *config.Config, e *echo.Echo) (*Store, error) {
	db, err := gorm.Open("postgres", fmt.Sprintf(
		"sslmode=disable host=%s port=%s user=%s dbname=%s password=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.DBName, cfg.DB.Password,
	))
	if err != nil {
		return nil, err
	}

	store = &Store{db}
	store.db.SetLogger(e.Logger)
	e.Logger.Debugf("set db debug mode to %v", cfg.DB.Debug)
	store.db.LogMode(
		cfg.DB.Debug,
	).CreateTable(
		&User{},
	).AutoMigrate(
		&User{},
	)

	return store, nil
}

func (s *Store) Shutdown() error {
	return s.db.Close()
}
