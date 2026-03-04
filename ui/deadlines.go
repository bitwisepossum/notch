package ui

import "github.com/bitwisepossum/notch/todo"

type deadlineFormat struct {
	label  string
	layout string // Go time layout used by time.Format/time.Parse
	hint   string // short example shown in prompts/errors
}

var deadlineFormats = []deadlineFormat{
	{label: "YYYY-MM-DD", layout: "2006-01-02", hint: "2026-03-04"},
	{label: "DD/MM/YYYY", layout: "02/01/2006", hint: "04/03/2026"},
	{label: "MM/DD/YYYY", layout: "01/02/2006", hint: "03/04/2026"},
	{label: "DD Mon YYYY", layout: "02 Jan 2006", hint: "04 Mar 2026"},
}

func deadlineFormatIdx(s todo.Settings) int {
	if s.DeadlineFormat == "" {
		return 0
	}
	for i, f := range deadlineFormats {
		if f.layout == s.DeadlineFormat {
			return i
		}
	}
	return 0
}

func deadlineLayout(s todo.Settings) string {
	return deadlineFormats[deadlineFormatIdx(s)].layout
}

func deadlineLabel(s todo.Settings) string {
	return deadlineFormats[deadlineFormatIdx(s)].label
}

func deadlineHint(s todo.Settings) string {
	return deadlineFormats[deadlineFormatIdx(s)].hint
}

