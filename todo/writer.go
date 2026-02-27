package todo

import (
	"io"
	"strings"
)

// Write serializes items as GFM checkbox Markdown to w.
func Write(w io.Writer, items []*Item) error {
	return writeItems(w, items, 0)
}

func writeItems(w io.Writer, items []*Item, depth int) error {
	indent := strings.Repeat("  ", depth)
	for _, item := range items {
		mark := " "
		if item.Done {
			mark = "x"
		}
		line := indent + "- [" + mark + "] " + item.Text + "\n"
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
		if len(item.Children) > 0 {
			if err := writeItems(w, item.Children, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}
