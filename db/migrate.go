package main

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/util"
	"github.com/rs/zerolog/log"
)

func init() {
	util.InitLogger()
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to load config")
	}

	db, err := db.New(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to connect to db")
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("unable to establish driver")
	}

	migration, err := migrate.NewWithDatabaseInstance("file://db/migrations", "sqlite3", driver)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to establish migration")
	}

	if err := migration.Up(); err != nil {
		if err != migrate.ErrNoChange {
			log.Fatal().Err(err).Msg("unable to migrate")
		} else {
			log.Info().Msg("ℹ️ no migration required")
		}
		return
	}

	log.Info().Msg("✅ migration complete")
}
