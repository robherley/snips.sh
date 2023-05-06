package renderer_test

import (
	"testing"

	"github.com/robherley/snips.sh/internal/renderer"
	"github.com/stretchr/testify/assert"
)

func TestToMarkdownSantization(t *testing.T) {
	tc := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic",
			in:   "# Hello World",
			want: "<div class=\"markdown\"><h1 id=\"hello-world\">Hello World</h1>\n</div>",
		},
		{
			name: "no script",
			in:   "<script>alert('hello world')</script>",
			want: "<div class=\"markdown\"></div>",
		},
		{
			name: "no iframe",
			in:   "<iframe src=\"https://github.com\"></iframe>",
			want: "<div class=\"markdown\"></div>",
		},
		{
			name: "syntax highlighting",
			in:   "```js\nconsole.log('hello world')\n```",
			want: "<div class=\"markdown\"><pre class=\"chroma\"><code><span class=\"line\"><span class=\"cl\"><span class=\"nx\">console</span><span class=\"p\">.</span><span class=\"nx\">log</span><span class=\"p\">(</span><span class=\"s1\">&#39;hello world&#39;</span><span class=\"p\">)</span>\n</span></span></code></pre></div>",
		},
		{
			name: "allow language- class on code blocks",
			in:   "```mermaid\ngraph LR\nA-->B\n```",
			want: "<div class=\"markdown\"><pre><code class=\"language-mermaid\">graph LR\nA--&gt;B\n</code></pre>\n</div>",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderer.ToMarkdown([]byte(tt.in))
			if err != nil {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, string(got))
		})
	}
}
