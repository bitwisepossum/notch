package todo

import (
	"bufio"
	"io"
	"regexp"
	"time"
)

var itemRe = regexp.MustCompile(`^(\s*)- \[([ xX])\] (.+)$`)

// suspectRe matches lines that contain a checkbox marker but don't satisfy
// itemRe — e.g. "[x] Missing dash" or "  [x] Wrong prefix". These are logged
// as errors to help diagnose hand-edited files that won't parse as expected.
var suspectRe = regexp.MustCompile(`\[[ xX]\]`)

// deadlineRe matches the deadline suffix: @YYYY-MM-DD
// The legacy Obsidian Tasks format (📅 YYYY-MM-DD) is also accepted on read.
var deadlineRe = regexp.MustCompile(`\s*(?:@|📅\s*)(\d{4}-\d{2}-\d{2})\s*$`)

// Parse reads Markdown from r and returns a slice of top-level Items.
// It recognizes GFM checkbox lines with 2-space indentation for nesting.
// Non-matching lines (headings, blanks) are silently skipped.
// Obsidian Tasks deadline suffixes (📅 YYYY-MM-DD) are parsed and stripped from Text.
//
// The second return value contains any lines that look like checkbox items
// (contain [ ] or [x]) but did not match the expected format. Callers may log
// these as warnings. Parse itself never returns a non-nil error for such lines;
// only scanner IO errors produce a non-nil error.
func Parse(r io.Reader) ([]*Item, []string, error) {
	scanner := bufio.NewScanner(r)

	// stack tracks the parent slice at each depth level.
	var root []*Item
	var suspect []string
	stack := []*[]*Item{&root}

	for scanner.Scan() {
		line := scanner.Text()
		m := itemRe.FindStringSubmatch(line)
		if m == nil {
			if suspectRe.MatchString(line) {
				suspect = append(suspect, line)
			}
			continue
		}

		// 2-space units; non-multiple indents (e.g. 1 or 3 spaces from an
		// external editor) truncate silently rather than erroring.
		depth := len(m[1]) / 2
		done := m[2] == "x" || m[2] == "X"
		text := m[3]

		// Extract and strip deadline suffix if present.
		var deadline time.Time
		if dm := deadlineRe.FindStringSubmatch(text); dm != nil {
			if t, err := time.Parse("2006-01-02", dm[1]); err == nil {
				deadline = t
			}
			text = deadlineRe.ReplaceAllString(text, "")
		}

		item := &Item{Text: text, Done: done, Deadline: deadline}

		// Ensure the stack is deep enough. If we jumped deeper than expected
		// (malformed indent), clamp to the deepest available parent.
		if depth >= len(stack) {
			depth = len(stack) - 1
		}

		// Trim the stack to this depth level + 1.
		stack = stack[:depth+1]

		// Append to the current parent.
		parent := stack[depth]
		*parent = append(*parent, item)

		// Push this item's children as the next depth level.
		stack = append(stack, &item.Children)
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return root, suspect, nil
}
