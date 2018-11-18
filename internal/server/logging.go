package server

import (
	"io"
	"os"
	"path"

	"github.com/labstack/gommon/log"
	"github.com/sintell/lisariy-server/internal/pkg/config"
)

func getLogWriter(cfg *config.Config) (io.Writer, error) {
	err := os.MkdirAll(path.Dir(cfg.Log.LogsPath), 0755)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(cfg.Log.LogsPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	writers := []io.Writer{}
	if cfg.Log.AlsoLogToStdOut {
		writers = append(writers, os.Stdout)
	}
	writers = append(writers, f)

	return io.MultiWriter(writers...), nil
}

func getLogLevel(cfg *config.Config) log.Lvl {
	switch cfg.Log.Level {
	case "DEBUG":
		return log.DEBUG
	case "INFO":
		return log.INFO
	default:
		return log.INFO
	}
}
