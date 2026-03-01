package todo

import (
	"bufio"
	"io"
	"regexp"
)

var itemRe = regexp.MustCompile(`^(\s*)- \[([ xX])\] (.+)$`)

// Parse reads Markdown from r and returns a slice of top-level Items.
// It recognizes GFM checkbox lines with 2-space indentation for nesting.
// Non-matching lines (headings, blanks) are silently skipped.
func Parse(r io.Reader) ([]*Item, error) {
	scanner := bufio.NewScanner(r)

	// stack tracks the parent slice at each depth level.
	var root []*Item
	stack := []*[]*Item{&root}

	for scanner.Scan() {
		line := scanner.Text()
		m := itemRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		// 2-space units; non-multiple indents (e.g. 1 or 3 spaces from an
		// external editor) truncate silently rather than erroring.
		depth := len(m[1]) / 2
		done := m[2] == "x" || m[2] == "X"
		text := m[3]

		item := &Item{Text: text, Done: done}

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
		return nil, err
	}
	return root, nil
}
