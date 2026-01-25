package renderer

import (
	"bytes"
	_ "embed"
	"sync"

	"github.com/alecthomas/chroma/v2"
)

var (
	//go:embed theme.xml
	theme []byte

	style     *chroma.Style
	styleOnce sync.Once
)

func GetStyle() *chroma.Style {
	styleOnce.Do(func() {
		style = chroma.MustNewXMLStyle(bytes.NewReader(theme))
	})
	return style
}
