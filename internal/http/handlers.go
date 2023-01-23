package http

import (
	"net/http"
	"strings"

	"github.com/robherley/snips.sh/internal/db"
	"github.com/rs/zerolog/log"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Healthy\n"))
}

func FileHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileID := strings.TrimPrefix(r.URL.Path, "/f/")

		file := db.File{}
		if err := database.First(&file, "id = ?", fileID).Error; err != nil {
			log.Error().Err(err).Msg("unable to lookup file")
			http.NotFound(w, r)
			return
		}

		if file.Private {
			log.Warn().Msg("attempted to access private file")
			http.NotFound(w, r)
		}

		w.WriteHeader(http.StatusOK)
		w.Write(file.Content)
	}
}
