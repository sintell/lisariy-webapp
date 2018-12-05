package server

import (
	"fmt"

	"github.com/antonlindstrom/pgstore"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

const secret = "SomeDifficultToGetInfo3Dloi7Hd3d"

func registerMiddlewares(e *echo.Echo, cfg *config.Config) {
	st, err := pgstore.NewPGStore(
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName), []byte(secret))
	if err != nil {
		panic(err)
	}
	st.Options.HttpOnly = true
	st.Options.MaxAge = 60 * 24 * 60 * 60
	st.Options.Path = "/"

	e.Use(
		middleware.Recover(),
		middleware.Logger(),
		session.Middleware(st),
		setupSession,
	)
}
