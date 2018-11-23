package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/sintell/lisariy-server/internal/pkg/config"
	"github.com/sintell/lisariy-server/internal/server"
)

var (
	confPath string
)

func init() {
	flag.StringVar(&confPath, "config", "config/config.json", "file that holds configuration for application")
}

func main() {
	flag.Parse()

	srv := server.New()
	cfg, err := config.New(confPath)
	if err != nil {
		srv.GetLogger().Fatal("can't read config:", err)
	}

	signalChan := make(chan os.Signal, 1)
	exitChan := make(chan int)

	go srv.Start(cfg.Access())

	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go func() {
		for {
			s := <-signalChan
			switch s {
			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				fallthrough
			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				srv.GetLogger().Info("got SIGINT/SIGTERM")

				srv.Stop()
				exitChan <- 0
			default:
				exitChan <- 1
			}
		}
	}()

	os.Exit(<-exitChan)
}
