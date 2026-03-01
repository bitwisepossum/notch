package todo

import (
	"fmt"
	"strings"
)

// SearchResult pairs a matched item with its path in the tree.
type SearchResult struct {
	Path []int
	Item *Item
}

// Item represents a single TODO entry, potentially with nested children.
type Item struct {
	Text     string
	Done     bool
	Children []*Item
}

// List represents a named collection of TODO items, backed by a Markdown file.
type List struct {
	Name  string // derived from filename (without .md)
	Items []*Item
}

// resolve navigates the item tree using path indices, returning the target item,
// a pointer to the parent slice containing it, and its index within that slice.
func (l *List) resolve(path []int) (*Item, *[]*Item, int, error) {
	if len(path) == 0 {
		return nil, nil, -1, fmt.Errorf("empty path")
	}

	items := &l.Items
	for _, idx := range path[:len(path)-1] {
		if idx < 0 || idx >= len(*items) {
			return nil, nil, -1, fmt.Errorf("invalid path: index %d out of range (len %d)", idx, len(*items))
		}
		items = &(*items)[idx].Children
	}

	last := path[len(path)-1]
	if last < 0 || last >= len(*items) {
		return nil, nil, -1, fmt.Errorf("invalid path: index %d out of range (len %d)", last, len(*items))
	}
	return (*items)[last], items, last, nil
}

// resolveParent returns the slice where a new child would be appended for the
// given parent path. An empty path returns the top-level items slice.
func (l *List) resolveParent(parentPath []int) (*[]*Item, error) {
	if len(parentPath) == 0 {
		return &l.Items, nil
	}
	item, _, _, err := l.resolve(parentPath)
	if err != nil {
		return nil, err
	}
	return &item.Children, nil
}

// Add appends a new item under the parent identified by parentPath.
// An empty parentPath adds a top-level item.
func (l *List) Add(parentPath []int, text string) {
	items, err := l.resolveParent(parentPath)
	if err != nil {
		return
	}
	*items = append(*items, &Item{Text: text})
}

// Remove deletes the item at path along with all its children.
func (l *List) Remove(path []int) error {
	_, items, idx, err := l.resolve(path)
	if err != nil {
		return err
	}
	*items = append((*items)[:idx], (*items)[idx+1:]...)
	return nil
}

// Edit changes the text of the item at path.
func (l *List) Edit(path []int, newText string) error {
	item, _, _, err := l.resolve(path)
	if err != nil {
		return err
	}
	item.Text = newText
	return nil
}

// Toggle flips the Done state of the item at path.
func (l *List) Toggle(path []int) error {
	item, _, _, err := l.resolve(path)
	if err != nil {
		return err
	}
	item.Done = !item.Done
	return nil
}

// Search returns all items whose text contains query (case-insensitive).
// Results are ordered depth-first. An empty query matches every item.
func (l *List) Search(query string) []SearchResult {
	query = strings.ToLower(query)
	var results []SearchResult
	var walk func(items []*Item, prefix []int)
	walk = func(items []*Item, prefix []int) {
		for i, item := range items {
			path := make([]int, len(prefix)+1)
			copy(path, prefix)
			path[len(prefix)] = i
			if strings.Contains(strings.ToLower(item.Text), query) {
				results = append(results, SearchResult{Path: path, Item: item})
			}
			walk(item.Children, path)
		}
	}
	walk(l.Items, nil)
	return results
}

// Rename changes the list's name.
func (l *List) Rename(newName string) {
	l.Name = newName
}

// Move relocates an item from one path to another.
// The item (with its children) is removed from `from` and inserted at `to`.
func (l *List) Move(from, to []int) error {
	// Extract the item from its current location.
	item, fromItems, fromIdx, err := l.resolve(from)
	if err != nil {
		return fmt.Errorf("invalid from path: %w", err)
	}
	*fromItems = append((*fromItems)[:fromIdx], (*fromItems)[fromIdx+1:]...)

	// Find the destination parent and insert.
	if len(to) == 0 {
		return fmt.Errorf("empty destination path")
	}
	destParentPath := to[:len(to)-1]
	destIdx := to[len(to)-1]

	toItems, err := l.resolveParent(destParentPath)
	if err != nil {
		return fmt.Errorf("invalid to path: %w", err)
	}

	if destIdx < 0 || destIdx > len(*toItems) {
		destIdx = len(*toItems)
	}

	// Insert at destIdx.
	*toItems = append((*toItems)[:destIdx], append([]*Item{item}, (*toItems)[destIdx:]...)...)
	return nil
}
