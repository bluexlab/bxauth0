package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bluexlab/bxauth0/pkg/auth0"

	"github.com/alecthomas/kingpin/v2"
	formatter "github.com/bluexlab/logrus-formatter"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const (
	appName = "server"
	appDesc = "BlueX Auth0 Server"
)

func main() {
	formatter.InitLogger()
	_ = godotenv.Load()

	app := kingpin.New(appName, appDesc).Version("0.1.0")
	app.HelpFlag.Short('h')

	configFlag := app.Flag("config", "Config file").Short('c').Default("server.yml").String()
	serverCommand := app.Command("server", "Run the server")
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	logrus.Infof("Loading config from %s", *configFlag)
	config, err := LoadConfig(*configFlag)
	if err != nil {
		logrus.Errorf("Failed to load config: %s", err)
		os.Exit(1)
	}

	switch cmd {
	case serverCommand.FullCommand():
		RunServer(config)
	}
}

func RunServer(config *Config) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer stop()

	logrus.Infof("Starting server on %s:%d", config.HTTP.Host, config.HTTP.Port)
	srv := auth0.NewServer(
		auth0.WithHostPort(config.HTTP.Host, config.HTTP.Port),
		auth0.WithEndpoint(config.Endpoint),
		auth0.WithClient(config.ClientID, config.ClientSecret, config.ClientEmail),
	)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Errorf("Failed to run server: %s", err)
			stop()
		}
	}()

	// listen for the interrupt signal.
	<-ctx.Done()
	logrus.Info("Shutting down server...")

	// restore default behavior on the interrupt signal.
	stop()
	logrus.Info("Shutting down gracefully, press Ctrl+C again to force")

	if err := srv.Stop(ctx); err != nil {
		logrus.Errorf("Failed to stop server: %s", err)
		os.Exit(1)
	}

	wg.Wait()
	logrus.Info("Server stopped")
}
