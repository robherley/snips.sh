package prompt

type KindSetMsg struct {
	Kind       Kind
	Breadcrumb string
}

type FeedbackMsg struct {
	Feedback string
	Finished bool
}

type SelectorInitMsg struct{}
