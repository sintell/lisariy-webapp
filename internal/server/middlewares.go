package server

import (
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

func registerMiddlewares(e *echo.Echo, cfg config.Config) {
	e.Use(
		middleware.Recover(),
		middleware.Logger(),
	)
}
