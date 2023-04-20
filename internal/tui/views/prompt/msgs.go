package prompt

type KindSetMsg struct {
	Kind Kind
}

type FeedbackMsg struct {
	Feedback string
	Finished bool
}

type SelectorInitMsg struct{}
