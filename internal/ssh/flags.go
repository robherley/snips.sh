package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

var (
	ErrFlagRequied = errors.New("flag required")
)

type UploadFlags struct {
	*flag.FlagSet

	Private   bool
	Extension string
}

func (uf *UploadFlags) Parse(out io.Writer, args []string) error {
	uf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	uf.SetOutput(out)

	uf.BoolVar(&uf.Private, "private", false, "only accessible via creator or signed urls (optional)")
	uf.StringVar(&uf.Extension, "ext", "", "set the file extension (optional)")

	if err := uf.FlagSet.Parse(args); err != nil {
		return err
	}

	uf.Extension = strings.TrimPrefix(strings.ToLower(uf.Extension), ".")

	return nil
}

type SignFlags struct {
	*flag.FlagSet

	TTL time.Duration
}

func (sf *SignFlags) Parse(out io.Writer, args []string) error {
	sf.FlagSet = flag.NewFlagSet("sign", flag.ContinueOnError)
	sf.SetOutput(out)

	sf.DurationVar(&sf.TTL, "ttl", 0, "lifetime of the signed url")

	if err := sf.FlagSet.Parse(args); err != nil {
		return err
	}

	if sf.TTL.Seconds() == 0 {
		return fmt.Errorf("%w: -ttl", ErrFlagRequied)
	}

	return nil
}
