package ssh

import (
	"flag"
	"strings"
	"time"
)

type UploadFlags struct {
	Private   bool
	TTL       *time.Duration
	Extension *string
}

func ParseUploadFlags(sesh Session) (*UploadFlags, error) {
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.SetOutput(sesh.Stderr())

	flags := &UploadFlags{}

	ttl := time.Duration(0)
	ext := ""

	flagset.BoolVar(&flags.Private, "private", false, "file only accessible via the creator's keys (optional)")
	flagset.DurationVar(&ttl, "ttl", 0, "set the time-to-live for the file (optional)")
	flagset.StringVar(&ext, "ext", "", "set the file extension (optional)")

	if err := flagset.Parse(sesh.Command()); err != nil {
		return nil, err
	}

	if ext != "" {
		ext = strings.TrimPrefix(strings.ToLower(ext), ".")
		if len(ext) > 255 {
			ext = ext[:255]
		}

		flags.Extension = &ext
	}

	if ttl > 0 {
		flags.TTL = &ttl
	}

	return flags, nil
}
