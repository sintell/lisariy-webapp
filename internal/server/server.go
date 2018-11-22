package server

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/labstack/gommon/log"

	"github.com/labstack/echo"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

var (
	pp *PicturesProcessor
)

// Server wraps echo web server providing convinient methods
// to start/stop/re-read config files
type Server struct {
	cfg *config.Config
	e   *echo.Echo
	str *Store
	pp  *PicturesProcessor
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
	s.cfg = cfg

	output, err := getLogWriter(s.cfg)
	if err != nil {
		s.e.Logger.Fatal(err)
	}
	s.e.Logger.SetLevel(getLogLevel(s.cfg))
	s.e.Logger.SetOutput(output)

	s.str, err = NewStore(s.cfg, s.e)
	if err != nil {
		s.e.Logger.Fatal(err)
	}

	s.pp = NewPicturesProcessor(s.e.Logger)
	pp = s.pp
	s.pp.Start()

	registerMiddlewares(s.e, s.cfg)
	registerHandlers(s.e)
	registerProtectedHandlers(s.e)

	s.e.Logger.Info("starting server with PID=", os.Getpid(), "on PORT=", cfg.WebApp.Port)

	err = s.e.Start(":" + cfg.WebApp.Port)
	if err != nil && err != http.ErrServerClosed {
		s.e.Logger.Fatalf("unexpected server crash: %s", err)
		return nil
	}

	return s
}

// Stop is a gracefull shutdown for echo server
func (s *Server) Stop() {
	s.e.Logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.WebApp.ShutdownTimeout*time.Second)
	defer cancel()
	if err := s.e.Shutdown(ctx); err != nil {
		s.e.Logger.Fatalf("error during webserver shutdown: %s")
	}
	<-ctx.Done()
	if err := s.str.Shutdown(); err != nil {
		s.e.Logger.Fatalf("error during store shutdown: %s", err)
	}
	s.pp.Stop()
}

// GetLogger allows access to server internall logger
func (s *Server) GetLogger() echo.Logger {
	return s.e.Logger
}
