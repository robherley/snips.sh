package snips_test

import (
	"regexp"
	"testing"

	"github.com/robherley/snips.sh/internal/snips"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKeyToken(t *testing.T) {
	token, hash, err := snips.NewAPIKeyToken()
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`^snips_[A-Z2-7]{52}$`), token)
	require.Equal(t, snips.HashAPIKeyToken(token), hash)
}
