package renderer

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var (
	htmlSanitizer = bluemonday.UGCPolicy().
		// allow element alignment
		AllowAttrs("align").Globally().
		// allow element width
		AllowAttrs("width").Globally().
		// allow language class on code blocks (used for mermaid.js diagrams)
		AllowAttrs("class").Matching(regexp.MustCompile(`^language-(.*)$`)).OnElements("code").
		// allow chroma class on pre (used for syntax highlighting)
		AllowAttrs("class").Matching(regexp.MustCompile(`chroma$`)).OnElements("pre").
		// allow chroma classes on span elements (used for syntax highlighting)
		AllowAttrs("class").Matching(chromaSpanClassRegex).OnElements("span")

	// ChromaSpanClassRegex is a regex that matches all the elements that can are rendered by chroma
	chromaSpanClassRegex = regexp.MustCompile(`^(` + strings.Join(chromaSpanClasses, "|") + `)$`)

	// ChromaSpanClasses is a list of all the elements that can are rendered by chrome
	// see web/static/css/chroma.css for styling
	chromaSpanClasses = []string{
		"x",
		"err",
		"cl",
		"lnlinks",
		"lntd",
		"lntable",
		"hl",
		"ln",
		"line",
		"k",
		"kc",
		"kd",
		"kn",
		"kp",
		"kr",
		"kt",
		"n",
		"na",
		"nb",
		"bp",
		"nc",
		"no",
		"nd",
		"ni",
		"ne",
		"nf",
		"fm",
		"nl",
		"nn",
		"nx",
		"py",
		"nt",
		"nv",
		"vc",
		"vg",
		"vi",
		"vm",
		"l",
		"ld",
		"s",
		"sa",
		"sb",
		"sc",
		"dl",
		"sd",
		"s2",
		"se",
		"sh",
		"si",
		"sx",
		"sr",
		"s1",
		"ss",
		"m",
		"mb",
		"mf",
		"mh",
		"mi",
		"il",
		"mo",
		"o",
		"ow",
		"p",
		"c",
		"ch",
		"cm",
		"c1",
		"cs",
		"cp",
		"cpf",
		"g",
		"gd",
		"ge",
		"gr",
		"gh",
		"gi",
		"go",
		"gp",
		"gs",
		"gu",
		"gt",
		"gl",
		"w",
	}
)
