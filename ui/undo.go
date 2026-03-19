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
	foldCopy := make(map[string]bool, len(m.ib.folded))
	for k, v := range m.ib.folded {
		foldCopy[k] = v
	}
	return snapshot{
		list:       m.ib.list.Clone(),
		itemCursor: m.ib.cursor,
		folded:     foldCopy,
	}
}

// pushUndo saves the current state onto the undo stack before a mutation.
// It also clears the redo stack so branching history is discarded.
func (m *Model) pushUndo() {
	m.ib.undoStack = append(m.ib.undoStack, m.takeSnapshot())
	if len(m.ib.undoStack) > maxUndoSize {
		m.ib.undoStack = m.ib.undoStack[1:]
	}
	m.ib.redoStack = m.ib.redoStack[:0]
}

// applySnapshot restores model state from a snapshot and persists to disk.
func (m *Model) applySnapshot(s snapshot) {
	m.ib.list = s.list
	m.ib.folded = s.folded
	m.saveFlash()
	m.rebuildFlat()
	m.ib.cursor = min(s.itemCursor, max(len(m.ib.flat)-1, 0))
	m.ib.scroll = clampScroll(m.ib.cursor, m.ib.scroll, m.visibleRows(), len(m.ib.flat))
}

// undo reverts to the previous state, pushing the current state onto redo.
// Returns false if there is nothing to undo.
func (m *Model) undo() bool {
	if len(m.ib.undoStack) == 0 {
		return false
	}
	m.ib.redoStack = append(m.ib.redoStack, m.takeSnapshot())
	s := m.ib.undoStack[len(m.ib.undoStack)-1]
	m.ib.undoStack = m.ib.undoStack[:len(m.ib.undoStack)-1]
	m.applySnapshot(s)
	return true
}

// redo re-applies a previously undone change, pushing current state onto undo.
// Returns false if there is nothing to redo.
func (m *Model) redo() bool {
	if len(m.ib.redoStack) == 0 {
		return false
	}
	m.ib.undoStack = append(m.ib.undoStack, m.takeSnapshot())
	s := m.ib.redoStack[len(m.ib.redoStack)-1]
	m.ib.redoStack = m.ib.redoStack[:len(m.ib.redoStack)-1]
	m.applySnapshot(s)
	return true
}
