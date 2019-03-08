package server

import (
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/labstack/echo"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

var (
	store       *Store
	ErrNotFound = pg.ErrNoRows
)

const (
	maxRetries       = 6
	dialTimeout      = 10 * time.Second
	keepAliveTimeout = 1 * time.Second
)

type Store struct {
	db *pg.DB
}

func NewStore(cfg *config.Config, e *echo.Echo) (*Store, error) {
	db := pg.Connect(&pg.Options{
		Addr:            cfg.DB.Host + ":" + cfg.DB.Port,
		Database:        cfg.DB.DBName,
		User:            cfg.DB.User,
		Password:        cfg.DB.Password,
		ApplicationName: "lisariy-app",
		DialTimeout:     dialTimeout,
	})
	store = &Store{db}

	pg.SetLogger(e.StdLogger)

	if cfg.DB.Debug {
		store.db.OnQueryProcessed(func(event *pg.QueryProcessedEvent) {
			query, err := event.FormattedQuery()
			if err != nil {
				e.Logger.Fatal(err)
			}

			e.Logger.Debugf("%s %s", time.Since(event.StartTime), query)
		})
	}

	for _, model := range []interface{}{&User{}, &Tag{}, &Picture{}, &PictureToTag{}} {
		err := store.db.CreateTable(model, &orm.CreateTableOptions{
			FKConstraints: true,
			IfNotExists:   true,
		})
		if err != nil {
			e.Logger.Fatal(err)
		}
	}

	return store, nil
}

func (s *Store) Shutdown() error {
	return s.db.Close()
}
