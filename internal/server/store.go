package server

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	// using form dialect here
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

var (
	store       *Store
	ErrNotFound = gorm.ErrRecordNotFound
)

const (
	maxRetries       = 6
	retryTimeout     = 10 * time.Second
	keepAliveTimeout = 1 * time.Second
)

type Store struct {
	db *gorm.DB
}

func initStoreWithRetry(cfg *config.Config, e *echo.Echo) error {
	tryNum := 0
	for {
		db, err := gorm.Open("postgres", fmt.Sprintf(
			"sslmode=disable host=%s port=%s user=%s dbname=%s password=%s",
			cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.DBName, cfg.DB.Password,
		))
		if err != nil {
			if tryNum <= maxRetries {
				e.Logger.Errorf("error connecting to DB, will retry: %d of %d", tryNum, maxRetries)
				tryNum++
				time.Sleep(retryTimeout)
				continue
			}
			return err
		}

		store = &Store{db}
		break
	}
	return nil
}

func startWatcher(cfg *config.Config, e *echo.Echo) {
	go (func() {
		for {
			if err := store.db.DB().Ping(); err != nil {
				e.Logger.Errorf("db connection lost, will retry: %s", err)
				break
			}
			time.Sleep(keepAliveTimeout)
		}
		if err := initStoreWithRetry(cfg, e); err != nil {
			e.Logger.Fatal(err)
			return
		}
		startWatcher(cfg, e)
	})()
}

func NewStore(cfg *config.Config, e *echo.Echo) (*Store, error) {
	if err := initStoreWithRetry(cfg, e); err != nil {
		return nil, err
	}
	startWatcher(cfg, e)

	store.db.SetLogger(e.Logger)
	e.Logger.Debugf("set db debug mode to %v", cfg.DB.Debug)
	store.db.LogMode(
		cfg.DB.Debug,
	).AutoMigrate(
		&User{},
		&Picture{},
	)

	e.Logger.Debug(store.db.GetErrors())

	return store, nil
}

func (s *Store) Shutdown() error {
	return s.db.Close()
}
