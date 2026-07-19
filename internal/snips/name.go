package snips

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// NameMaxLength is the maximum length of a file's name.
	NameMaxLength = 40
)

var (
	ErrInvalidName = fmt.Errorf("names must be 1-%d alphanumeric characters (hyphen, dot, or underscore separators allowed)", NameMaxLength)

	// nameRegex matches alphanumeric words separated by single hyphens, dots,
	// or underscores. Requiring the name to start and end with an alphanumeric
	// rune also rules out "." and ".." path segments.
	nameRegex = regexp.MustCompile(`^[a-zA-Z0-9]+(?:[-._][a-zA-Z0-9]+)*$`)
)

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" || len(name) > NameMaxLength {
		return "", ErrInvalidName
	}

	if !nameRegex.MatchString(name) {
		return "", ErrInvalidName
	}

	return name, nil
}
