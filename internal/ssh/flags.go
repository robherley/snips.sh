package ssh

import (
	"flag"
	"strings"
)

type UploadFlags struct {
	Private   bool
	Extension *string
}

func ParseUploadFlags(sesh Session) (*UploadFlags, error) {
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	flagset.SetOutput(sesh.Stderr())

	flags := &UploadFlags{}
	ext := ""

	flagset.BoolVar(&flags.Private, "private", false, "file only accessible via the creator's keys (optional)")
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

	return flags, nil
}
