package todo

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"
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
	Deadline time.Time // zero = no deadline
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

// SetDeadline sets or clears the deadline on the item at path.
// Pass a zero time.Time to clear an existing deadline.
func (l *List) SetDeadline(path []int, d time.Time) error {
	item, _, _, err := l.resolve(path)
	if err != nil {
		return err
	}
	item.Deadline = d
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

// ToggleCascade flips Done on the item at path, then sets all descendants to match.
func (l *List) ToggleCascade(path []int) error {
	item, _, _, err := l.resolve(path)
	if err != nil {
		return err
	}
	item.Done = !item.Done
	setDoneRecursive(item.Children, item.Done)
	return nil
}

// setDoneRecursive sets Done on all items in the slice and their descendants.
func setDoneRecursive(items []*Item, done bool) {
	for _, item := range items {
		item.Done = done
		setDoneRecursive(item.Children, done)
	}
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

// ChildCount returns the number of direct children at parentPath.
// An empty parentPath returns the top-level item count.
func (l *List) ChildCount(parentPath []int) int {
	items, err := l.resolveParent(parentPath)
	if err != nil {
		return 0
	}
	return len(*items)
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

// Clone returns a deep copy of the list, duplicating all items and children.
// Used by the UI undo/redo system to snapshot list state before mutations.
func (l *List) Clone() *List {
	return &List{
		Name:  l.Name,
		Items: cloneItems(l.Items),
	}
}

// cloneItems recursively copies a slice of items.
func cloneItems(items []*Item) []*Item {
	if len(items) == 0 {
		return nil
	}
	out := make([]*Item, len(items))
	for i, item := range items {
		out[i] = &Item{
			Text:     item.Text,
			Done:     item.Done,
			Deadline: item.Deadline,
			Children: cloneItems(item.Children),
		}
	}
	return out
}

// Hash returns a short fingerprint of the list's structure and content,
// covering item texts, done states, and hierarchy. Used to detect whether
// persisted fold state is still valid for the current list.
func (l *List) Hash() string {
	h := fnv.New64a()
	var walk func(items []*Item, depth int)
	walk = func(items []*Item, depth int) {
		for _, item := range items {
			fmt.Fprintf(h, "%d|%v|%s|%s\n", depth, item.Done, item.Text, item.Deadline.Format("2006-01-02"))
			walk(item.Children, depth+1)
		}
	}
	walk(l.Items, 0)
	return fmt.Sprintf("%x", h.Sum64())
}
