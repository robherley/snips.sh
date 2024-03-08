package apiv1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
)

// GetFeed
// Retrieve a list of snips on the server, paginated, in date created DESC by
// default.
// Ideal utilisation, when trying to view all snips, is to iterate through pages
// until you reach a payload size of 0.
func GetFeed(database db.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		page := 0
		if r.URL.Query().Has("page") {
			strPage := r.URL.Query().Get("page")
			p, err := strconv.Atoi(strPage)
			if err == nil {
				page = p
			}
		}

		files, err := database.LatestPublicFiles(r.Context(), page, 10)
		if err != nil {
			log.Error().Err(err).Msg("unable to get latest files")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		filesMarshalled, err := json.Marshal(files)
		if err != nil {
			log.Error().Err(err).Msg("unable to encode json")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(filesMarshalled)
	}
}

func GetFile(database db.DB) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log := logger.From(r.Context())

		file, err := database.FindFile(r.Context(), chi.URLParam(r, "file-id"))
		if err != nil {
			log.Error().Err(err).Msg("unable to get file")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		if file == nil || file.Private {
			http.NotFound(w, r)
			return
		}

		filesMarshalled, err := json.Marshal(file)
		if err != nil {
			log.Error().Err(err).Msg("unable to render template")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(filesMarshalled)
	}
}

func APIHandler(database db.DB) *chi.Mux {
	apiRouter := chi.NewMux()

	apiRouter.Get("/feed", GetFeed(database))
	apiRouter.Get("/files/{file-id}", GetFile(database))

	return apiRouter
}
