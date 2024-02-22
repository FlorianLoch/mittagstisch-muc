package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	kongyaml "github.com/alecthomas/kong-yaml"
	"github.com/florianloch/mittagstisch/internal/config"
	"github.com/florianloch/mittagstisch/internal/database"
	"github.com/florianloch/mittagstisch/internal/handler"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	envKeyConfig = "MITTAGSTISCH_CONFIG_PATH"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	cli := &struct {
		Config   config.Config `embed:""`
		ServeCmd ServeCmd      `cmd:"" default:"1" help:"Default command"`
	}{}

	kong.ConfigureHelp(kong.HelpOptions{Compact: false, Summary: true})

	kong.Name("mittagstisch")
	kong.Description("Backend for a portal showing places offering a business lunch.")

	configPath := os.Getenv(envKeyConfig)

	if configPath == "" {
		configPath = "./mittagstisch.config.yaml"
	}

	loader := kong.ConfigurationLoader(func(r io.Reader) (kong.Resolver, error) {
		input, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("reading config: %w", err)
		}

		if err := yaml.Unmarshal(input, &cli.Config); err != nil {
			return nil, fmt.Errorf("parsing topic mapping: %w", err)
		}

		return kongyaml.Loader(bytes.NewReader(input))
	})

	ktx := kong.Parse(cli, kong.Configuration(loader, configPath))
	if ktx.Error != nil {
		log.Fatal().Err(ktx.Error).Msg("Failed to parse input parameters/commands")
	}

	cfg := &cli.Config

	if cfg.Verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Debug().Interface("cfg", cfg).Msg("Loaded config")

	if err := ktx.Run(cfg); err != nil {
		log.Fatal().Err(err).Msg("Failed to run")
	}
}

type ServeCmd struct{}

func (s *ServeCmd) Run(cfg *config.Config) error {
	ctx, cancelFn := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancelFn()

	//log.Info().Msg("Starting server...")
	//
	//spaHandler, err := webapp.New(webapp.WebappFS, "dist", "app/", "index.html")
	//if err != nil {
	//	return fmt.Errorf("failed to create SPA handler: %w", err)
	//}

	timeoutCtx, cancelFn := context.WithTimeout(ctx, 1*time.Minute)
	defer cancelFn()

	db, err := database.New(timeoutCtx, cfg.DBConnString)
	if err != nil {
		return fmt.Errorf("setting up database: %w", err)
	}

	controller := handler.NewController(*cfg, db, &log.Logger)

	srv := http.Server{
		Addr:    cfg.ListenAddr,
		Handler: controller,
	}
	serveWG := &sync.WaitGroup{}
	serveWG.Add(2)

	go func() {
		defer serveWG.Done()

		<-ctx.Done()

		log.Debug().Msg("Shutdown signal received. Shutting down server...")

		timeout, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelFn()

		if err := srv.Shutdown(timeout); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown server gracefully.")
		}
	}()

	go func() {
		defer serveWG.Done()

		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed running server.")
		}
	}()

	log.Info().Msgf("Server is ready to handle requests at http://%s", cfg.ListenAddr)

	<-ctx.Done()

	log.Info().Msg("Waiting for connections to be closed and for server to shutdown...")

	serveWG.Wait()

	log.Info().Msg("Server has been shut down. Bye.")

	return nil
}
