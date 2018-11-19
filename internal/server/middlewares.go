package server

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
	"github.com/sintell/lisariy-server/internal/pkg/config"
	"github.com/wader/gormstore"
)

const secret = "SomeDifficultToGetInfo3Dloi7Hd3d"

func registerMiddlewares(e *echo.Echo, cfg *config.Config) {
	st := gormstore.NewOptions(
		store.db,
		gormstore.Options{SkipCreateTable: false},
		[]byte(secret),
	)
	st.SessionOpts.HttpOnly = true
	st.SessionOpts.MaxAge = 60 * 24 * 60 * 60
	st.SessionOpts.Path = "/"

	e.Use(
		middleware.Recover(),
		middleware.Logger(),
		session.Middleware(st),
		setupSession,
	)
}
