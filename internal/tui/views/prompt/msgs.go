package prompt

import "github.com/robherley/snips.sh/internal/tui/feedback"

type KindSetMsg struct {
	Kind       Kind
	Breadcrumb string
}

type FeedbackMsg struct {
	Feedback feedback.Feedback
	Finished bool
}

type SelectorInitMsg struct{}
