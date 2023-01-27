package ssh

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

var (
	ErrFlagRequied = errors.New("flag required")
)

type UploadFlags struct {
	*flag.FlagSet

	Private   bool
	Extension *string
}

func (uf *UploadFlags) Parse(sesh Session) error {
	uf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	uf.SetOutput(sesh.Stderr())

	ext := ""

	uf.BoolVar(&uf.Private, "private", false, "only accessible via creator or signed urls (optional)")
	uf.StringVar(&ext, "ext", "", "set the file extension (optional)")

	if err := uf.FlagSet.Parse(sesh.Command()); err != nil {
		return err
	}

	if ext != "" {
		ext = strings.TrimPrefix(strings.ToLower(ext), ".")
		if len(ext) > 255 {
			ext = ext[:255]
		}

		uf.Extension = &ext
	}

	return nil
}

type SignFlags struct {
	*flag.FlagSet

	TTL time.Duration
}

func (sf *SignFlags) Parse(sesh Session) error {
	sf.FlagSet = flag.NewFlagSet("sign", flag.ContinueOnError)
	sf.SetOutput(sesh.Stderr())

	sf.DurationVar(&sf.TTL, "ttl", 0, "lifetime of the signed url")

	if err := sf.FlagSet.Parse(sesh.Command()[1:]); err != nil {
		return err
	}

	if sf.TTL.Seconds() == 0 {
		return fmt.Errorf("%w: -ttl", ErrFlagRequied)
	}

	return nil
}
