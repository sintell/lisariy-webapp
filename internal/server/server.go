package server

import (
	"context"
	"os"
	"time"

	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"

	"github.com/labstack/echo"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

// Server wraps echo web server providing convinient methods
// to start/stop/re-read config files
type Server struct {
	cfg *config.Config
	e   *echo.Echo
}

// New creates new instance of webapp server
func New() *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetLevel(log.DEBUG)
	return &Server{e: e}
}

// Start internaly starts echo web-server
// and register all handlers
func (s *Server) Start(cfg *config.Config) *Server {
	s.e.Logger.Info("starting server with PID=", os.Getpid(), "on PORT=", cfg.WebApp.Port)
	s.cfg = cfg

	output, err := getLogWriter(s.cfg)
	if err != nil {
		s.e.Logger.Fatal(err)
	}
	s.e.Logger.SetLevel(getLogLevel(s.cfg))
	s.e.Logger.SetOutput(output)

	s.e.Group("api", middleware.RemoveTrailingSlash())
	s.e.Logger.Fatal(s.e.Start(":" + cfg.WebApp.Port))

	return s
}

// Stop is a gracefull shutdown for echo server
func (s *Server) Stop() {
	s.e.Logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.WebApp.ShutdownTimeout*time.Second)
	defer cancel()
	if err := s.e.Shutdown(ctx); err != nil {
		s.e.Logger.Fatal(err)
	}
	<-ctx.Done()
}

// GetLogger allows access to server internall logger
func (s *Server) GetLogger() echo.Logger {
	return s.e.Logger
}
