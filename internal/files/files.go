// Package files holds file mutations shared by the SSH and HTTP frontends.
package files

import (
	"context"
	"fmt"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/robherley/snips.sh/internal/config"
	"github.com/robherley/snips.sh/internal/db"
	"github.com/robherley/snips.sh/internal/logger"
	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/robherley/snips.sh/internal/snips"
)

// UpdateContent replaces a file's content and persists it, re-detecting the
// file type (optionally hinted by extension) and recording a revision diff
// for non-binary files. Revision bookkeeping failures are logged, not fatal.
func UpdateContent(ctx context.Context, database *db.DB, cfg *config.Config, file *snips.File, content []byte, extension string) error {
	log := logger.From(ctx)

	file.Size = uint64(len(content))
	file.Type = renderer.DetectFileType(content, extension, cfg.EnableGuesser)

	// Compute diff for revision history (skip binary files)
	if !file.IsBinary() {
		oldContent, err := database.Files.GetContent(ctx, file.ID)
		if err != nil {
			log.Warn("unable to get old content for diff", "err", err)
		} else {
			revCount, err := database.Revisions.CountByFileID(ctx, file.ID)
			if err != nil {
				log.Warn("unable to count revisions", "err", err)
			}
			fromLabel := fmt.Sprintf("%s (v%d)", file.ID, revCount)
			toLabel := fmt.Sprintf("%s (v%d)", file.ID, revCount+1)
			diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(oldContent)),
				B:        difflib.SplitLines(string(content)),
				FromFile: fromLabel,
				ToFile:   toLabel,
				Context:  3,
			})
			if err != nil {
				log.Warn("unable to compute diff", "err", err)
			} else if diff != "" {
				revision := &snips.Revision{
					FileID: file.ID,
					Size:   file.Size,
					Type:   file.Type,
				}
				if err := database.Revisions.Create(ctx, revision, []byte(diff), cfg.FileCompression, cfg.Limits.RevisionsPerFile); err != nil {
					log.Warn("unable to create revision", "err", err)
				}
			}
		}
	}

	return database.Files.UpdateContent(ctx, file, content, cfg.FileCompression)
}
