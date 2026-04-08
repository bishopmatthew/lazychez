package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Rect represents a bounding rectangle in terminal coordinates.
type Rect struct {
	X, Y, W, H int
}

// Contains returns true if (x, y) falls within this rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// PaneRects holds bounding rectangles for all panes, computed on-the-fly.
type PaneRects struct {
	FileList  Rect
	GitStatus Rect
	Status    Rect
	Diff      Rect
	MaxY      int  // coordinates at y >= MaxY are misses (footer/status bar)
	Narrow    bool // whether we're in narrow layout
}

// layoutGeometry computes pane bounding rects from current dimensions and focus state.
// Used by both View() (indirectly via distribute helpers) and hit-test.
// Wide layout stacking order (left column): FileList, GitStatus, Status
// Narrow layout stacking order: FileList, GitStatus, Status, Diff
func (m Model) layoutGeometry() PaneRects {
	contentHeight := m.height - 2
	if contentHeight < 0 {
		contentHeight = 0
	}
	maxY := contentHeight

	if m.isNarrow() {
		statusH, fileH, gitH, diffH := m.distributeNarrow(contentHeight)
		y := 0
		fileRect := Rect{0, y, m.width, fileH}
		y += fileH
		gitRect := Rect{0, y, m.width, gitH}
		y += gitH
		statusRect := Rect{0, y, m.width, statusH}
		y += statusH
		diffRect := Rect{0, y, m.width, diffH}
		return PaneRects{
			FileList:  fileRect,
			GitStatus: gitRect,
			Status:    statusRect,
			Diff:      diffRect,
			MaxY:      maxY,
			Narrow:    true,
		}
	}

	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth
	statusH, fileH, gitH := m.distributeLeftColumn(contentHeight)

	y := 0
	fileRect := Rect{0, y, leftWidth, fileH}
	y += fileH
	gitRect := Rect{0, y, leftWidth, gitH}
	y += gitH
	statusRect := Rect{0, y, leftWidth, statusH}
	diffRect := Rect{leftWidth, 0, rightWidth, contentHeight}

	return PaneRects{
		FileList:  fileRect,
		GitStatus: gitRect,
		Status:    statusRect,
		Diff:      diffRect,
		MaxY:      maxY,
	}
}

// HitTarget describes what a mouse coordinate maps to.
type HitTarget struct {
	Pane     PaneID
	RowIndex int  // logical row index within the pane (accounting for scroll offset), -1 if not a row
	IsMiss   bool // true if the coordinate is outside all panes
}

// hitTest maps a terminal coordinate to a pane target using the given geometry.
// Pure function — no model dependencies beyond the geometry.
func hitTest(x, y int, rects PaneRects) HitTarget {
	if y >= rects.MaxY {
		return HitTarget{IsMiss: true}
	}

	// Check each pane in z-order (all at same z-level for non-overlay).
	// In narrow layout, collapsed panes (H == collapsedHeight) are still valid targets.
	type candidate struct {
		pane PaneID
		rect Rect
	}
	panes := []candidate{
		{PaneFileList, rects.FileList},
		{PaneGitStatus, rects.GitStatus},
		{PaneStatus, rects.Status},
		{PaneInfo, rects.Diff},
	}

	for _, c := range panes {
		if c.rect.Contains(x, y) {
			rowIndex := -1
			// Compute row index for list panes (not collapsed, not border)
			if c.pane == PaneFileList || c.pane == PaneGitStatus {
				if !rects.Narrow || c.rect.H > collapsedHeight {
					// visualRow is relative to inner content (subtract top border)
					visualRow := y - c.rect.Y - 1
					innerHeight := c.rect.H - paneChrome
					if visualRow >= 0 && visualRow < innerHeight {
						rowIndex = visualRow
					}
				}
			}
			return HitTarget{Pane: c.pane, RowIndex: rowIndex}
		}
	}

	return HitTarget{IsMiss: true}
}

// overlayRect computes the bounding rectangle of the current overlay by
// rendering it and measuring the result. Returns the rect and true if an
// overlay is active. The render cost is negligible since this only runs on
// mouse events.
func (m Model) overlayRect() (Rect, bool) {
	var rendered string
	switch m.overlay {
	case OverlayHelp:
		rendered = m.renderHelp()
	case OverlayCommit:
		rendered = m.renderCommitInput()
	case OverlayConfirmApply:
		rendered = m.renderConfirmApply()
	case OverlayConfirmReAdd:
		rendered = m.renderConfirmReAdd()
	case OverlayConfirmApplyAll:
		rendered = m.renderConfirmApplyAll()
	case OverlayConfirmGitDiscard:
		rendered = m.renderConfirmGitDiscard()
	case OverlayConfirmForget:
		rendered = m.renderConfirmForget()
	case OverlayAddFile:
		rendered = m.renderAddFileOverlay()
	case OverlayConfirmStageAll:
		rendered = m.renderConfirmStageAll()
	default:
		return Rect{}, false
	}

	w := lipgloss.Width(rendered)
	h := lipgloss.Height(rendered)
	x := (m.width - w) / 2
	y := (m.height - h) / 2

	return Rect{x, y, w, h}, true
}

// handleMouse processes a mouse event and returns the updated model and command.
func (m Model) handleMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	// Zero-dimension guard: no geometry available before first WindowSizeMsg.
	if m.width == 0 || m.height == 0 {
		return m, nil
	}

	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	// Overlay intercepts all mouse events (z-order: overlay > panes).
	if m.overlay != OverlayNone {
		return m.handleOverlayMouse(msg)
	}

	// Scroll wheel events: dispatch to pane under cursor without changing focus.
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		return m.handleScroll(msg)
	}

	// Only handle left-click. Middle-click and right-click are silently ignored.
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	rects := m.layoutGeometry()
	target := hitTest(msg.X, msg.Y, rects)

	if target.IsMiss {
		return m, nil
	}

	// Always focus the clicked pane.
	focusChanged := target.Pane != m.focused
	if focusChanged {
		m.setFocus(target.Pane)
		m.updateDimensions()
	}

	// Click-to-select in list panes.
	switch target.Pane {
	case PaneFileList:
		return m.handleFileListClick(target, rects)
	case PaneGitStatus:
		return m.handleGitStatusClick(target, rects)
	}

	// Non-list panes: focus change only.
	if focusChanged {
		return m, m.fetchDiffForFocusedPane()
	}
	return m, nil
}

// handleOverlayMouse routes mouse events when an overlay is open.
// Clicks outside the overlay dismiss it. Clicks/scrolls inside route
// to overlay-specific handlers.
func (m Model) handleOverlayMouse(msg tea.MouseMsg) (Model, tea.Cmd) {
	rect, ok := m.overlayRect()
	if !ok {
		return m, nil
	}

	inside := rect.Contains(msg.X, msg.Y)

	// Scroll events inside overlay.
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		if !inside {
			return m, nil
		}
		up := msg.Button == tea.MouseButtonWheelUp
		return m.handleOverlayScroll(up)
	}

	// Only handle left-click.
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	// Click outside dismisses the overlay without interacting with background panes.
	if !inside {
		if m.overlay == OverlayCommit {
			m.commitInput.Blur()
		}
		m.overlay = OverlayNone
		return m, nil
	}

	// Click inside — handle per overlay type.
	switch m.overlay {
	case OverlayAddFile:
		return m.handleAddFileOverlayClick(msg, rect)
	}

	// Help, Commit, Confirm overlays: interior click is a no-op.
	return m, nil
}

// handleOverlayScroll processes scroll events inside an overlay.
func (m Model) handleOverlayScroll(up bool) (Model, tea.Cmd) {
	switch m.overlay {
	case OverlayHelp:
		if up {
			m.helpViewport.LineUp(1)
		} else {
			m.helpViewport.LineDown(1)
		}
	case OverlayAddFile:
		if up {
			m.addFile.moveUp()
		} else {
			m.addFile.moveDown()
		}
	}
	return m, nil
}

// handleAddFileOverlayClick handles click-to-toggle in the AddFile overlay.
// OverlayStyle chrome: border(1) + padding(top:1, left:2).
// AddFile content layout: row 0 = title, row 1 = status, rows 2..N = file items.
func (m Model) handleAddFileOverlayClick(msg tea.MouseMsg, rect Rect) (Model, tea.Cmd) {
	const (
		overlayBorderX    = 1
		overlayPaddingX   = 2
		overlayBorderY    = 1
		overlayPaddingY   = 1
		contentHeaderRows = 2 // title + status line
	)

	innerY := msg.Y - rect.Y - overlayBorderY - overlayPaddingY
	fileRow := innerY - contentHeaderRows

	rows := m.addFile.rows()
	if fileRow < 0 || fileRow >= rows {
		return m, nil
	}

	// Determine column from X position.
	innerX := msg.X - rect.X - overlayBorderX - overlayPaddingX
	col := 0
	cw := m.addFile.colWidth()
	if m.addFile.columns > 1 && cw > 0 && innerX >= cw+2 {
		col = 1
	}

	// Column-first layout: offset + col*rows + row.
	idx := m.addFile.offset + col*rows + fileRow
	if idx >= len(m.addFile.filtered) {
		return m, nil
	}

	m.addFile.cursor = idx
	m.addFile.ToggleSelected()
	return m, nil
}

// handleScroll processes scroll wheel events with scroll-under-cursor routing.
// Scrolling dispatches to the pane under the mouse, regardless of m.focused.
// Scrolling does NOT change pane focus.
// Scroll over collapsed pane bars in narrow layout is a no-op.
func (m Model) handleScroll(msg tea.MouseMsg) (Model, tea.Cmd) {
	rects := m.layoutGeometry()
	target := hitTest(msg.X, msg.Y, rects)

	if target.IsMiss {
		return m, nil
	}

	// In narrow layout, ignore scroll over collapsed pane bars.
	if rects.Narrow {
		var h int
		switch target.Pane {
		case PaneFileList:
			h = rects.FileList.H
		case PaneGitStatus:
			h = rects.GitStatus.H
		case PaneStatus:
			h = rects.Status.H
		case PaneInfo:
			h = rects.Diff.H
		}
		if h <= collapsedHeight {
			return m, nil
		}
	}

	up := msg.Button == tea.MouseButtonWheelUp

	switch target.Pane {
	case PaneFileList:
		return m.handleFileListScroll(up)
	case PaneGitStatus:
		return m.handleGitStatusScroll(up)
	case PaneInfo:
		return m.handleDiffScroll(up)
	}

	return m, nil
}

// handleFileListScroll moves the FileList cursor by one item on scroll.
func (m Model) handleFileListScroll(up bool) (Model, tea.Cmd) {
	prevPath := m.fileList.SelectedPath()

	if up {
		m.fileList.MoveUp()
	} else {
		m.fileList.MoveDown()
	}

	newPath := m.fileList.SelectedPath()
	if newPath != "" && newPath != prevPath {
		m.catMode = false
		m.syncFocus()
		if diff, ok := m.diffCache[newPath]; ok {
			m.diffView.SetContent(newPath, diff)
			return m, nil
		}
		return m, fetchDiff(m.chezmoi, newPath)
	}

	return m, nil
}

// handleGitStatusScroll moves the GitStatus cursor by one item on scroll.
func (m Model) handleGitStatusScroll(up bool) (Model, tea.Cmd) {
	prevPath := m.gitStatus.SelectedPath()

	if up {
		m.gitStatus.MoveUp()
	} else {
		m.gitStatus.MoveDown()
	}

	newPath := m.gitStatus.SelectedPath()
	if newPath != "" && newPath != prevPath {
		return m, fetchGitDiff(m.git, newPath)
	}

	return m, nil
}

// handleDiffScroll scrolls the diff viewport by one line on scroll.
// Calls viewport.LineUp/LineDown directly, bypassing DiffViewModel.Update()
// which guards on m.focused — enabling scroll-under-cursor.
func (m Model) handleDiffScroll(up bool) (Model, tea.Cmd) {
	if !m.diffView.ready {
		return m, nil
	}
	if up {
		m.diffView.viewport.LineUp(1)
	} else {
		m.diffView.viewport.LineDown(1)
	}
	return m, nil
}

// handleFileListClick handles click-to-select in the FileList pane.
func (m Model) handleFileListClick(target HitTarget, rects PaneRects) (Model, tea.Cmd) {
	if target.RowIndex < 0 {
		return m, m.fetchDiffForFocusedPane()
	}

	logicalIndex := target.RowIndex + m.fileList.offset
	if logicalIndex >= len(m.fileList.files) {
		return m, m.fetchDiffForFocusedPane()
	}

	// Heading rows are ignored for selection.
	if m.fileList.files[logicalIndex].IsHeading {
		return m, m.fetchDiffForFocusedPane()
	}

	// During FilterTyping, lock the filter first.
	if m.fileList.IsFiltering() {
		m.fileList.LockFilter()
		// Recalculate since LockFilter resets cursor.
		if logicalIndex >= len(m.fileList.files) {
			return m, m.fetchDiffForSelected()
		}
		if m.fileList.files[logicalIndex].IsHeading {
			return m, m.fetchDiffForSelected()
		}
	}

	prevPath := m.fileList.SelectedPath()
	m.fileList.SetCursor(logicalIndex)
	newPath := m.fileList.SelectedPath()

	if newPath != "" && newPath != prevPath {
		m.catMode = false
		m.syncFocus()
		if diff, ok := m.diffCache[newPath]; ok {
			m.diffView.SetContent(newPath, diff)
			return m, nil
		}
		return m, fetchDiff(m.chezmoi, newPath)
	}

	return m, nil
}

// handleGitStatusClick handles click-to-select in the GitStatus pane.
func (m Model) handleGitStatusClick(target HitTarget, rects PaneRects) (Model, tea.Cmd) {
	if target.RowIndex < 0 {
		return m, m.fetchDiffForFocusedPane()
	}

	logicalIndex := target.RowIndex + m.gitStatus.offset
	if logicalIndex >= len(m.gitStatus.entries) {
		return m, m.fetchDiffForFocusedPane()
	}

	prevPath := m.gitStatus.SelectedPath()
	m.gitStatus.SetCursor(logicalIndex)
	newPath := m.gitStatus.SelectedPath()

	if newPath != "" && newPath != prevPath {
		return m, fetchGitDiff(m.git, newPath)
	}

	return m, nil
}
