package renderer

import "html/template"

var BinaryHTMLPlaceholder = template.HTML(`
<div style="margin:2rem;text-align:center;">
		<span role="img" aria-label="warning">⚠️</span>
		The file is not displayed because it has been detected as binary data.
</div>
`)
