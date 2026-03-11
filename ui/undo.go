package ui

import "github.com/bitwisepossum/notch/todo"

const maxUndoSize = 50

// snapshot captures enough model state to fully restore a previous revision.
type snapshot struct {
	list       *todo.List
	itemCursor int
	folded     map[string]bool
}

// takeSnapshot returns a deep copy of the current undo-relevant model state.
func (m *Model) takeSnapshot() snapshot {
	foldCopy := make(map[string]bool, len(m.folded))
	for k, v := range m.folded {
		foldCopy[k] = v
	}
	return snapshot{
		list:       m.list.Clone(),
		itemCursor: m.itemCursor,
		folded:     foldCopy,
	}
}

// pushUndo saves the current state onto the undo stack before a mutation.
// It also clears the redo stack so branching history is discarded.
func (m *Model) pushUndo() {
	m.undoStack = append(m.undoStack, m.takeSnapshot())
	if len(m.undoStack) > maxUndoSize {
		m.undoStack = m.undoStack[1:]
	}
	m.redoStack = m.redoStack[:0]
}

// applySnapshot restores model state from a snapshot and persists to disk.
func (m *Model) applySnapshot(s snapshot) {
	m.list = s.list
	m.folded = s.folded
	m.saveFlash()
	m.rebuildFlat()
	m.itemCursor = min(s.itemCursor, max(len(m.flat)-1, 0))
	m.itemScroll = clampScroll(m.itemCursor, m.itemScroll, m.visibleRows(), len(m.flat))
}

// undo reverts to the previous state, pushing the current state onto redo.
// Returns false if there is nothing to undo.
func (m *Model) undo() bool {
	if len(m.undoStack) == 0 {
		return false
	}
	m.redoStack = append(m.redoStack, m.takeSnapshot())
	s := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]
	m.applySnapshot(s)
	return true
}

// redo re-applies a previously undone change, pushing current state onto undo.
// Returns false if there is nothing to redo.
func (m *Model) redo() bool {
	if len(m.redoStack) == 0 {
		return false
	}
	m.undoStack = append(m.undoStack, m.takeSnapshot())
	s := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]
	m.applySnapshot(s)
	return true
}
