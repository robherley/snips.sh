package prompt

type Kind int

const (
	None Kind = iota
	ChangeExtension
	ChangeVisibility
	GenerateSignedURL
	DeleteFile
)

// title is the modal breadcrumb segment for the prompt kind, matching the
// option names in the browser's options modal.
func (k Kind) title() string {
	switch k {
	case ChangeExtension:
		return "edit extension"
	case ChangeVisibility:
		return "toggle visibility"
	case GenerateSignedURL:
		return "generate signed url"
	case DeleteFile:
		return "delete file"
	default:
		return ""
	}
}
