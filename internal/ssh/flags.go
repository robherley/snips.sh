package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/robherley/snips.sh/internal/timeutil"
)

var (
	ErrFlagRequired = errors.New("flag required")
	ErrFlagParse    = errors.New("parse error")
)

type UploadFlags struct {
	*flag.FlagSet

	Private   bool
	Extension string
	TTL       time.Duration
	Name      string
}

func (uf *UploadFlags) Parse(out io.Writer, args []string) error {
	uf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	uf.SetOutput(out)

	uf.BoolVar(&uf.Private, "private", false, "only accessible via creator or signed urls (optional)")
	uf.StringVar(&uf.Extension, "ext", "", "set the file extension (optional)")
	addDurationFlag(uf.FlagSet, &uf.TTL, "ttl", 0, "lifetime of the signed url (optional)")
	uf.StringVar(&uf.Name, "name", "", "human-readable name for the file, must be unique per user (optional)")

	if err := uf.FlagSet.Parse(args); err != nil {
		return err
	}

	if uf.TTL.Seconds() > 0 && !uf.Private {
		return fmt.Errorf("%w: -private", ErrFlagRequired)
	}

	return nil
}

type APIKeyCreateFlags struct {
	*flag.FlagSet

	Name string
	TTL  time.Duration
}

func (af *APIKeyCreateFlags) Parse(out io.Writer, args []string) error {
	af.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	af.SetOutput(out)

	af.StringVar(&af.Name, "name", "", "human-readable name for the api key")
	addDurationFlag(af.FlagSet, &af.TTL, "ttl", 0, "lifetime of the api key (optional, never expires when omitted)")

	if err := af.FlagSet.Parse(args); err != nil {
		return err
	}

	if af.Name == "" {
		return fmt.Errorf("%w: -name", ErrFlagRequired)
	}
	if af.TTL < 0 {
		return fmt.Errorf("%w: -ttl", ErrFlagParse)
	}

	return nil
}

type RenameFlags struct {
	*flag.FlagSet

	Remove bool
}

func (rf *RenameFlags) Parse(out io.Writer, args []string) error {
	rf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	rf.SetOutput(out)

	rf.BoolVar(&rf.Remove, "rm", false, "remove the file's name")

	return rf.FlagSet.Parse(args)
}

type SignFlags struct {
	*flag.FlagSet

	TTL time.Duration
}

func (sf *SignFlags) Parse(out io.Writer, args []string) error {
	sf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	sf.SetOutput(out)

	addDurationFlag(sf.FlagSet, &sf.TTL, "ttl", 0, "lifetime of the signed url")

	if err := sf.FlagSet.Parse(args); err != nil {
		return err
	}

	if sf.TTL.Seconds() == 0 {
		return fmt.Errorf("%w: -ttl", ErrFlagRequired)
	}

	return nil
}

type DeleteFlags struct {
	*flag.FlagSet

	Force bool
}

func (df *DeleteFlags) Parse(out io.Writer, args []string) error {
	df.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	df.SetOutput(out)

	df.BoolVar(&df.Force, "f", false, "force delete without confirmation")

	return df.FlagSet.Parse(args)
}

type UpdateFileContentFlags struct {
	*flag.FlagSet

	Extension string
}

func (uf *UpdateFileContentFlags) Parse(out io.Writer, args []string) error {
	uf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	uf.SetOutput(out)

	uf.StringVar(&uf.Extension, "ext", "", "hint the file extension (optional)")

	return uf.FlagSet.Parse(args)
}

// durationFlagValue is a wrapper around time.Duration that implements the flag.Value interface using a custom parser.
type durationFlagValue time.Duration

// addDurationFlag adds a flag for a time.Duration to the given flag.FlagSet.
func addDurationFlag(fs *flag.FlagSet, p *time.Duration, name string, value time.Duration, usage string) {
	*p = value
	fs.Var((*durationFlagValue)(p), name, usage)
}

// Set implements the flag.Value interface.
func (d *durationFlagValue) Set(s string) error {
	v, err := timeutil.ParseDuration(s)
	if err != nil {
		err = ErrFlagParse
	}
	*d = durationFlagValue(v)
	return err
}

// Get implements the flag.Getter interface.
func (d *durationFlagValue) Get() any {
	return time.Duration(*d)
}

// String implements the flag.Value interface.
func (d *durationFlagValue) String() string {
	return (*time.Duration)(d).String()
}
