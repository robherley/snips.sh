package ssh

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/robherley/snips.sh/internal/timeutil"
)

var (
	ErrFlagRequied = errors.New("flag required")
	ErrFlagParse   = errors.New("parse error")
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

	uf.StringVar(&uf.Name, "name", "", "define custom name, if unavailable then a unique hash will be appended")
	uf.BoolVar(&uf.Private, "private", false, "only accessible via creator or signed urls (optional)")
	uf.StringVar(&uf.Extension, "ext", "", "set the file extension (optional)")
	addDurationFlag(uf.FlagSet, &uf.TTL, "ttl", 0, "lifetime of the signed url (optional)")

	if err := uf.FlagSet.Parse(args); err != nil {
		return err
	}

	if uf.TTL.Seconds() > 0 && !uf.Private {
		return fmt.Errorf("%w: -private", ErrFlagRequied)
	}

	uf.Extension = strings.TrimPrefix(strings.ToLower(uf.Extension), ".")
	uf.Name = strings.Trim(uf.Name, " ")

	return nil
}

type RenameFlags struct {
	*flag.FlagSet

	Name string
}

func (sf *RenameFlags) Parse(out io.Writer, args []string) error {
	sf.FlagSet = flag.NewFlagSet("", flag.ContinueOnError)
	sf.SetOutput(out)

	addStringFlag(sf.FlagSet, &sf.Name, "name", "", "define custom name, if unavailable then a unique hash will be appended")

	if err := sf.FlagSet.Parse(args); err != nil {
		return err
	}

	return nil
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
		return fmt.Errorf("%w: -ttl", ErrFlagRequied)
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

// stringFlagValue is a wrapper around string that implements the flag.Value interface using a custom parser.
type stringFlagValue string

// addStringFlag adds a flag for a string to the given flag.FlagSet.
func addStringFlag(fs *flag.FlagSet, flagValue *string, name string, value string, usage string) {
	*flagValue = value
	fs.Var((*stringFlagValue)(flagValue), name, usage)
}

// Set implements the flag.Value interface.
func (sf *stringFlagValue) Set(s string) error {
	*sf = stringFlagValue(s)
	return nil
}

// Get implements the flag.Getter interface.
func (sf *stringFlagValue) Get() any {
	return string(*sf)
}

// String implements the flag.Value interface.
func (sf *stringFlagValue) String() string {
	return string(*sf)
}
