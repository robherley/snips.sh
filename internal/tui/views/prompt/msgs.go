package prompt

type PromptKindSetMsg struct {
	Kind Kind
}

type PromptFeedbackMsg struct {
	Feedback string
	Finished bool
}
