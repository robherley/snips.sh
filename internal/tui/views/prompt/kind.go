package prompt

type Kind int

const (
	None Kind = iota
	ChangeExtension
	ChangeVisibility
	GenerateSignedURL
	DeleteFile
	ChangeName
	ChangeDescription
)
