package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLayoutGeometry_WideLayout(t *testing.T) {
	m, _, _ := newTestModel() // 120x40, wide layout

	rects := m.layoutGeometry()
	contentHeight := m.height - 2 // 38

	if rects.Narrow {
		t.Error("expected wide layout, got narrow")
	}

	leftWidth := m.width / 3
	rightWidth := m.width - leftWidth

	// Verify diff pane position
	if rects.Diff.X != leftWidth {
		t.Errorf("Diff.X = %d, want %d", rects.Diff.X, leftWidth)
	}
	if rects.Diff.W != rightWidth {
		t.Errorf("Diff.W = %d, want %d", rects.Diff.W, rightWidth)
	}
	if rects.Diff.H != contentHeight {
		t.Errorf("Diff.H = %d, want %d", rects.Diff.H, contentHeight)
	}

	// Verify left column starts at x=0
	if rects.FileList.X != 0 || rects.GitStatus.X != 0 || rects.Status.X != 0 {
		t.Error("left column panes should start at x=0")
	}

	// Verify left column widths
	if rects.FileList.W != leftWidth {
		t.Errorf("FileList.W = %d, want %d", rects.FileList.W, leftWidth)
	}

	// Verify MaxY
	if rects.MaxY != contentHeight {
		t.Errorf("MaxY = %d, want %d", rects.MaxY, contentHeight)
	}
}

func TestLayoutGeometry_Invariant_NoOverlap_SumEqualsHeight(t *testing.T) {
	tests := []struct {
		name    string
		width   int
		height  int
		focused PaneID
	}{
		{"wide default", 120, 40, PaneFileList},
		{"wide git focused", 120, 40, PaneGitStatus},
		{"wide status focused", 120, 40, PaneStatus},
		{"wide diff focused", 120, 40, PaneInfo},
		{"narrow file focused", 60, 40, PaneFileList},
		{"narrow git focused", 60, 40, PaneGitStatus},
		{"narrow status focused", 60, 40, PaneStatus},
		{"small terminal", 85, 15, PaneFileList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel()
			m.width = tt.width
			m.height = tt.height
			m.focused = tt.focused
			m.updateDimensions()

			rects := m.layoutGeometry()
			contentHeight := tt.height - 2

			if rects.Narrow {
				// Narrow: all panes stacked vertically, sum of heights == contentHeight
				sum := rects.FileList.H + rects.GitStatus.H + rects.Status.H + rects.Diff.H
				if sum != contentHeight {
					t.Errorf("narrow height sum = %d, want %d", sum, contentHeight)
				}

				// Verify no overlap: each pane starts where the previous ends
				if rects.GitStatus.Y != rects.FileList.Y+rects.FileList.H {
					t.Error("GitStatus.Y doesn't follow FileList")
				}
				if rects.Status.Y != rects.GitStatus.Y+rects.GitStatus.H {
					t.Error("Status.Y doesn't follow GitStatus")
				}
				if rects.Diff.Y != rects.Status.Y+rects.Status.H {
					t.Error("Diff.Y doesn't follow Status")
				}
			} else {
				// Wide: left column heights sum to contentHeight
				leftSum := rects.FileList.H + rects.GitStatus.H + rects.Status.H
				if leftSum != contentHeight {
					t.Errorf("wide left column height sum = %d, want %d", leftSum, contentHeight)
				}

				// Verify no vertical overlap in left column
				if rects.GitStatus.Y != rects.FileList.Y+rects.FileList.H {
					t.Error("GitStatus.Y doesn't follow FileList")
				}
				if rects.Status.Y != rects.GitStatus.Y+rects.GitStatus.H {
					t.Error("Status.Y doesn't follow GitStatus")
				}

				// Diff pane spans full content height
				if rects.Diff.H != contentHeight {
					t.Errorf("Diff.H = %d, want %d", rects.Diff.H, contentHeight)
				}

				// No horizontal overlap between left and right
				if rects.FileList.X+rects.FileList.W > rects.Diff.X {
					t.Error("left column overlaps diff pane horizontally")
				}
			}
		})
	}
}

func TestHitTest_WideLayout(t *testing.T) {
	m, _, _ := newTestModel() // 120x40
	rects := m.layoutGeometry()

	leftWidth := m.width / 3
	contentHeight := m.height - 2
	statusH, fileH, _ := m.distributeLeftColumn(contentHeight)

	tests := []struct {
		name     string
		x, y     int
		wantPane PaneID
		wantMiss bool
	}{
		// FileList pane
		{"file list interior", 5, 3, PaneFileList, false},
		{"file list top-left corner", 0, 0, PaneFileList, false},
		{"file list right border", leftWidth - 1, 0, PaneFileList, false},

		// GitStatus pane
		{"git status interior", 5, fileH + 2, PaneGitStatus, false},
		{"git status top border", 0, fileH, PaneGitStatus, false},

		// Status pane
		{"status pane", 5, fileH + (contentHeight - fileH - statusH) + 1, PaneStatus, false},

		// Diff pane
		{"diff pane interior", leftWidth + 5, 5, PaneInfo, false},
		{"diff pane left border", leftWidth, 0, PaneInfo, false},
		{"diff pane right edge", m.width - 1, 5, PaneInfo, false},

		// Footer area (miss)
		{"footer area", 5, contentHeight, 0, true},
		{"below screen", 5, m.height, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := hitTest(tt.x, tt.y, rects)
			if target.IsMiss != tt.wantMiss {
				t.Errorf("IsMiss = %v, want %v", target.IsMiss, tt.wantMiss)
			}
			if !tt.wantMiss && target.Pane != tt.wantPane {
				t.Errorf("Pane = %d, want %d", target.Pane, tt.wantPane)
			}
		})
	}
}

func TestHitTest_BoundaryPixels(t *testing.T) {
	m, _, _ := newTestModel() // 120x40
	rects := m.layoutGeometry()

	leftWidth := m.width / 3

	// Last pixel of left column → left pane
	target := hitTest(leftWidth-1, 0, rects)
	if target.IsMiss || target.Pane == PaneInfo {
		t.Errorf("x=%d should be in left column, got Pane=%d IsMiss=%v", leftWidth-1, target.Pane, target.IsMiss)
	}

	// First pixel of right column → diff pane
	target = hitTest(leftWidth, 0, rects)
	if target.IsMiss || target.Pane != PaneInfo {
		t.Errorf("x=%d should be diff pane, got Pane=%d IsMiss=%v", leftWidth, target.Pane, target.IsMiss)
	}

	// Last content row → still a pane
	contentHeight := m.height - 2
	target = hitTest(0, contentHeight-1, rects)
	if target.IsMiss {
		t.Errorf("y=%d should be in a pane, got miss", contentHeight-1)
	}

	// First footer row → miss
	target = hitTest(0, contentHeight, rects)
	if !target.IsMiss {
		t.Errorf("y=%d should be a miss (footer), got Pane=%d", contentHeight, target.Pane)
	}
}

func TestHitTest_RowIndex(t *testing.T) {
	m, _, _ := newTestModel() // 120x40
	rects := m.layoutGeometry()

	// Click on top border of FileList (y=0) → rowIndex should be -1
	target := hitTest(5, 0, rects)
	if target.RowIndex != -1 {
		t.Errorf("top border click: RowIndex = %d, want -1", target.RowIndex)
	}

	// Click on first content row of FileList (y=1, accounting for top border)
	target = hitTest(5, 1, rects)
	if target.RowIndex != 0 {
		t.Errorf("first content row: RowIndex = %d, want 0", target.RowIndex)
	}

	// Click on second content row
	target = hitTest(5, 2, rects)
	if target.RowIndex != 1 {
		t.Errorf("second content row: RowIndex = %d, want 1", target.RowIndex)
	}

	// Click on diff pane → no row index
	leftWidth := m.width / 3
	target = hitTest(leftWidth+5, 5, rects)
	if target.RowIndex != -1 {
		t.Errorf("diff pane click: RowIndex = %d, want -1", target.RowIndex)
	}
}

func TestHitTest_ZeroDimensionGuard(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 0
	m.height = 0

	mouseMsg := tea.MouseMsg{
		X:      10,
		Y:      10,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if cmd != nil {
		t.Error("expected nil cmd for zero dimensions")
	}
	// Should return early without panic
	if result.width != 0 {
		t.Error("model should be unchanged")
	}
}

func TestHandleMouse_ClickToFocus(t *testing.T) {
	m, _, _ := newTestModel() // 120x40, starts focused on PaneFileList
	rects := m.layoutGeometry()

	// Click on diff pane should focus it
	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.focused != PaneInfo {
		t.Errorf("after clicking diff: focused = %d, want %d", result.focused, PaneInfo)
	}
}

func TestHandleMouse_ClickSamePaneNoChange(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneFileList

	// Click on FileList (already focused) — should not trigger focus change
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.focused != PaneFileList {
		t.Errorf("focused changed unexpectedly to %d", result.focused)
	}
	if cmd != nil {
		t.Error("expected nil cmd when clicking already-focused pane")
	}
}

func TestHandleMouse_MiddleRightClickIgnored(t *testing.T) {
	m, _, _ := newTestModel()
	rects := m.layoutGeometry()

	for _, btn := range []tea.MouseButton{tea.MouseButtonMiddle, tea.MouseButtonRight} {
		mouseMsg := tea.MouseMsg{
			X:      rects.Diff.X + 5,
			Y:      5,
			Button: btn,
			Action: tea.MouseActionPress,
		}

		result, cmd := m.handleMouse(mouseMsg)
		if result.focused != PaneFileList {
			t.Errorf("button %d: focus changed to %d, should remain %d", btn, result.focused, PaneFileList)
		}
		if cmd != nil {
			t.Errorf("button %d: expected nil cmd", btn)
		}
	}
}

func TestHandleMouse_FooterClickMiss(t *testing.T) {
	m, _, _ := newTestModel()
	contentHeight := m.height - 2

	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      contentHeight, // footer area
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.focused != PaneFileList {
		t.Errorf("footer click changed focus to %d", result.focused)
	}
	if cmd != nil {
		t.Error("expected nil cmd for footer click")
	}
}

func TestLayoutGeometry_GeometryFreshness(t *testing.T) {
	m, _, _ := newTestModel() // 120x40

	rects1 := m.layoutGeometry()

	// Simulate a WindowSizeMsg changing dimensions
	m.width = 200
	m.height = 50
	m.updateDimensions()

	rects2 := m.layoutGeometry()

	if rects1.Diff.W == rects2.Diff.W {
		t.Error("geometry should change after resize")
	}
	if rects2.MaxY != 48 { // 50 - 2
		t.Errorf("MaxY = %d, want 48", rects2.MaxY)
	}
}

func TestHandleMouse_ClickBorderFocusesPane(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneFileList
	rects := m.layoutGeometry()

	// Click on the top border of the diff pane (y=0, x=leftWidth)
	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.focused != PaneInfo {
		t.Errorf("border click: focused = %d, want %d", result.focused, PaneInfo)
	}
}

// --- Phase 2: Click-to-select tests ---

func TestHandleMouse_ClickToSelectFileList(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: 'M', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click the third content row (y = FileList.Y + 1 border + 2 rows = row index 2)
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 2, // top border + 2 rows = third row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 2 {
		t.Errorf("cursor = %d, want 2", result.fileList.cursor)
	}
	if result.focused != PaneFileList {
		t.Errorf("focused = %d, want PaneFileList", result.focused)
	}
	// Should trigger diff fetch for new file
	if cmd == nil {
		t.Error("expected non-nil cmd for diff fetch")
	}
}

func TestHandleMouse_ClickToSelectGitStatus(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneGitStatus
	m.gitStatus.SetEntries([]GitStatusEntry{
		{XY: "M ", Path: "file1.go"},
		{XY: "??", Path: "file2.go"},
		{XY: " M", Path: "file3.go"},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click second row in GitStatus pane
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.GitStatus.Y + 1 + 1, // top border + 1 row = second row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.gitStatus.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.gitStatus.cursor)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for git diff fetch")
	}
}

func TestHandleMouse_ClickHeadingIgnored(t *testing.T) {
	m, _, _ := newTestModel()
	// Create a list with a heading at index 1
	m.fileList.files = []FileItem{
		{Path: ".bashrc", AddCol: 'M', ApplyCol: ' '},
		{IsHeading: true},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	}
	m.fileList.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click the heading row (index 1)
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 1, // top border + 1 = second row (heading)
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Cursor should not have moved to the heading
	if result.fileList.cursor != 0 {
		t.Errorf("cursor moved to %d, should stay at 0 (heading click ignored)", result.fileList.cursor)
	}
}

func TestHandleMouse_ClickBelowLastItem(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click far below the single item (visual row 10, but only 1 item)
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 10,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Should focus pane but not change cursor
	if result.fileList.cursor != 0 {
		t.Errorf("cursor = %d, should remain 0", result.fileList.cursor)
	}
}

func TestHandleMouse_ClickEmptyPane(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneFileList
	// GitStatus has no entries
	m.gitStatus.SetEntries(nil)
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click inside GitStatus pane
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.GitStatus.Y + 2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Should focus GitStatus
	if result.focused != PaneGitStatus {
		t.Errorf("focused = %d, want PaneGitStatus", result.focused)
	}
}

func TestHandleMouse_ClickWithScrollOffset(t *testing.T) {
	m, _, _ := newTestModel()
	// Create many files so the list is scrolled
	files := make([]FileItem, 30)
	for i := range files {
		files[i] = FileItem{Path: fmt.Sprintf("file%02d.go", i), AddCol: ' ', ApplyCol: ' '}
	}
	m.fileList.SetFiles(files)
	// Scroll down so offset > 0
	m.fileList.cursor = 20
	m.fileList.clampOffset()
	m.updateDimensions()

	rects := m.layoutGeometry()
	scrollOffset := m.fileList.offset

	// Click the first visible row
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1, // top border + first content row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Should select logical index = scrollOffset + 0
	if result.fileList.cursor != scrollOffset {
		t.Errorf("cursor = %d, want %d (scroll offset)", result.fileList.cursor, scrollOffset)
	}
}

func TestHandleMouse_CrossPaneClickSelectsRow(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneGitStatus // Start focused on git

	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click the second row in FileList while focused on GitStatus
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 1, // second content row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	// Should switch focus AND select the row
	if result.focused != PaneFileList {
		t.Errorf("focused = %d, want PaneFileList", result.focused)
	}
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.fileList.cursor)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for diff fetch on cross-pane click")
	}
}

func TestHandleMouse_FilterTypingClickLocksAndSelects(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.StartFilter()
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click the second row during filter typing
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 1,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Filter should be locked
	if result.fileList.filterMode != FilterLocked {
		t.Errorf("filterMode = %d, want FilterLocked", result.fileList.filterMode)
	}
	// Cursor should be on the clicked row
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.fileList.cursor)
	}
}

func TestHandleMouse_FilterLockedClickSelectsFromFilteredList(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.filterMode = FilterLocked
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click the third row in the filtered list
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 2 {
		t.Errorf("cursor = %d, want 2", result.fileList.cursor)
	}
}

func TestHandleMouse_ReleaseIgnored(t *testing.T) {
	m, _, _ := newTestModel()
	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.focused != PaneFileList {
		t.Errorf("release event changed focus to %d", result.focused)
	}
	if cmd != nil {
		t.Error("expected nil cmd for release event")
	}
}

// --- Phase 3: Scroll wheel in list panes tests ---

func TestHandleMouse_ScrollFileListDown(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.fileList.cursor)
	}
	// Should trigger diff fetch for new selection
	if cmd == nil {
		t.Error("expected non-nil cmd for diff fetch after scroll")
	}
}

func TestHandleMouse_ScrollFileListUp(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.cursor = 2
	m.updateDimensions()

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2,
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.fileList.cursor)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for diff fetch after scroll")
	}
}

func TestHandleMouse_ScrollGitStatusDown(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneGitStatus
	m.gitStatus.SetEntries([]GitStatusEntry{
		{XY: "M ", Path: "file1.go"},
		{XY: "??", Path: "file2.go"},
		{XY: " M", Path: "file3.go"},
	})
	m.gitStatus.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.GitStatus.Y + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.gitStatus.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.gitStatus.cursor)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for diff fetch after scroll")
	}
}

func TestHandleMouse_ScrollUnderCursorDoesNotChangeFocus(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneInfo // Focus on diff pane
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Scroll over FileList while focused on diff — should NOT change focus
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.focused != PaneInfo {
		t.Errorf("scroll changed focus to %d, should remain PaneInfo", result.focused)
	}
	// But cursor should have moved
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.fileList.cursor)
	}
}

func TestHandleMouse_ScrollAtBoundsNoop(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Scroll up at top — should be a no-op
	m.fileList.cursor = 0
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2,
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 0 {
		t.Errorf("scroll up at top: cursor = %d, want 0", result.fileList.cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd for scroll at bounds")
	}

	// Scroll down at bottom — should be a no-op
	m.fileList.cursor = 1
	mouseMsg.Button = tea.MouseButtonWheelDown

	result, cmd = m.handleMouse(mouseMsg)
	if result.fileList.cursor != 1 {
		t.Errorf("scroll down at bottom: cursor = %d, want 1", result.fileList.cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd for scroll at bounds")
	}
}

func TestHandleMouse_ScrollFileListSkipsHeadings(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.files = []FileItem{
		{Path: ".bashrc", AddCol: 'M', ApplyCol: ' '},
		{IsHeading: true},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	}
	m.fileList.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Scroll down should skip the heading at index 1
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (should skip heading at index 1)", result.fileList.cursor)
	}
}

func TestHandleMouse_ScrollDuringFilterTyping(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.StartFilter()
	m.fileList.cursor = 0
	m.updateDimensions()

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (scroll during filter typing)", result.fileList.cursor)
	}
	// Filter should still be in typing mode
	if result.fileList.filterMode != FilterTyping {
		t.Errorf("filterMode = %d, want FilterTyping", result.fileList.filterMode)
	}
}

func TestHandleMouse_ScrollFooterMiss(t *testing.T) {
	m, _, _ := newTestModel()
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
	})
	m.fileList.cursor = 0
	m.updateDimensions()

	contentHeight := m.height - 2

	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      contentHeight, // footer area
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 0 {
		t.Errorf("scroll on footer moved cursor to %d", result.fileList.cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd for scroll on footer")
	}
}

// --- Phase 4: Scroll wheel in diff pane tests ---

func TestHandleMouse_ScrollDiffDown(t *testing.T) {
	m, _, _ := newTestModel()
	// Load diff content long enough to scroll
	lines := ""
	for i := 0; i < 100; i++ {
		lines += fmt.Sprintf("+line %d\n", i)
	}
	m.diffView.SetDimensions(80, 20)
	m.diffView.SetContent("test.go", lines)

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.diffView.viewport.YOffset != 1 {
		t.Errorf("YOffset = %d, want 1", result.diffView.viewport.YOffset)
	}
	if cmd != nil {
		t.Error("expected nil cmd for diff scroll")
	}
}

func TestHandleMouse_ScrollDiffUp(t *testing.T) {
	m, _, _ := newTestModel()
	lines := ""
	for i := 0; i < 100; i++ {
		lines += fmt.Sprintf("+line %d\n", i)
	}
	m.diffView.SetDimensions(80, 20)
	m.diffView.SetContent("test.go", lines)
	// Scroll down first so we can scroll up
	m.diffView.viewport.LineDown(5)

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.diffView.viewport.YOffset != 4 {
		t.Errorf("YOffset = %d, want 4", result.diffView.viewport.YOffset)
	}
}

func TestHandleMouse_ScrollDiffUnderCursorDoesNotChangeFocus(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneFileList // Focus on FileList, scroll over diff
	lines := ""
	for i := 0; i < 100; i++ {
		lines += fmt.Sprintf("+line %d\n", i)
	}
	m.diffView.SetDimensions(80, 20)
	m.diffView.SetContent("test.go", lines)

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.focused != PaneFileList {
		t.Errorf("scroll over diff changed focus to %d, should remain PaneFileList", result.focused)
	}
	if result.diffView.viewport.YOffset != 1 {
		t.Errorf("YOffset = %d, want 1 (diff should have scrolled)", result.diffView.viewport.YOffset)
	}
}

func TestHandleMouse_ScrollDiffAtTopNoop(t *testing.T) {
	m, _, _ := newTestModel()
	lines := ""
	for i := 0; i < 100; i++ {
		lines += fmt.Sprintf("+line %d\n", i)
	}
	m.diffView.SetDimensions(80, 20)
	m.diffView.SetContent("test.go", lines)
	// Already at top

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.diffView.viewport.YOffset != 0 {
		t.Errorf("YOffset = %d, want 0 (already at top)", result.diffView.viewport.YOffset)
	}
}

func TestHandleMouse_ScrollDiffEmptyNoop(t *testing.T) {
	m, _, _ := newTestModel()
	m.diffView.SetDimensions(80, 20)
	// No content set — rawDiff is empty

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.diffView.viewport.YOffset != 0 {
		t.Errorf("YOffset = %d, want 0 (empty diff)", result.diffView.viewport.YOffset)
	}
	if cmd != nil {
		t.Error("expected nil cmd for empty diff scroll")
	}
}

func TestHandleMouse_ScrollDiffNotReady(t *testing.T) {
	m, _, _ := newTestModel()
	// Don't call SetDimensions — viewport is not ready
	m.diffView.ready = false

	rects := m.layoutGeometry()

	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	_ = result // Should not panic
	if cmd != nil {
		t.Error("expected nil cmd for not-ready diff scroll")
	}
}

// --- Phase 5: Overlay mouse interaction tests ---

func TestOverlayRect_NoOverlay(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayNone

	_, ok := m.overlayRect()
	if ok {
		t.Error("overlayRect should return false when no overlay is active")
	}
}

func TestOverlayRect_HelpOverlay(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayHelp
	m.initHelpViewport()

	rect, ok := m.overlayRect()
	if !ok {
		t.Fatal("overlayRect should return true for help overlay")
	}

	// Overlay should be centered
	if rect.X <= 0 || rect.Y <= 0 {
		t.Errorf("overlay should be centered, got X=%d Y=%d", rect.X, rect.Y)
	}
	if rect.W <= 0 || rect.H <= 0 {
		t.Errorf("overlay should have positive dimensions, got W=%d H=%d", rect.W, rect.H)
	}
	// Should fit within screen
	if rect.X+rect.W > m.width || rect.Y+rect.H > m.height {
		t.Errorf("overlay exceeds screen bounds: rect=%+v screen=%dx%d", rect, m.width, m.height)
	}
}

func TestHandleMouse_OverlayClickOutsideDismisses(t *testing.T) {
	overlays := []struct {
		name    string
		overlay OverlayMode
		setup   func(m *Model)
	}{
		{"help", OverlayHelp, func(m *Model) { m.initHelpViewport() }},
		{"commit", OverlayCommit, func(m *Model) {
			m.commitInput.Focus()
		}},
		{"confirm apply", OverlayConfirmApply, nil},
		{"confirm re-add", OverlayConfirmReAdd, nil},
		{"confirm apply all", OverlayConfirmApplyAll, nil},
		{"confirm discard", OverlayConfirmGitDiscard, nil},
		{"confirm forget", OverlayConfirmForget, nil},
		{"confirm stage all", OverlayConfirmStageAll, nil},
		{"add file", OverlayAddFile, func(m *Model) {
			w := min(100, max(40, m.width*80/100))
			h := min(30, max(10, m.height*70/100))
			m.addFile = NewAddFileModel([]string{"file1", "file2"}, w-4, h)
		}},
	}

	for _, tt := range overlays {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel()
			m.overlay = tt.overlay
			if tt.setup != nil {
				tt.setup(&m)
			}

			// Click at (0, 0) — top-left corner, outside any centered overlay
			mouseMsg := tea.MouseMsg{
				X:      0,
				Y:      0,
				Button: tea.MouseButtonLeft,
				Action: tea.MouseActionPress,
			}

			result, _ := m.handleMouse(mouseMsg)
			if result.overlay != OverlayNone {
				t.Errorf("overlay = %d, want OverlayNone after click outside", result.overlay)
			}
		})
	}
}

func TestHandleMouse_OverlayClickInsideDoesNotDismiss(t *testing.T) {
	overlays := []struct {
		name    string
		overlay OverlayMode
		setup   func(m *Model)
	}{
		{"help", OverlayHelp, func(m *Model) { m.initHelpViewport() }},
		{"commit", OverlayCommit, func(m *Model) {
			m.commitInput.Focus()
		}},
		{"confirm apply", OverlayConfirmApply, nil},
		{"confirm re-add", OverlayConfirmReAdd, nil},
		{"confirm apply all", OverlayConfirmApplyAll, nil},
		{"confirm discard", OverlayConfirmGitDiscard, nil},
		{"confirm forget", OverlayConfirmForget, nil},
		{"confirm stage all", OverlayConfirmStageAll, nil},
	}

	for _, tt := range overlays {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel()
			m.overlay = tt.overlay
			if tt.setup != nil {
				tt.setup(&m)
			}

			rect, ok := m.overlayRect()
			if !ok {
				t.Fatal("overlayRect should return true")
			}

			// Click in the center of the overlay
			mouseMsg := tea.MouseMsg{
				X:      rect.X + rect.W/2,
				Y:      rect.Y + rect.H/2,
				Button: tea.MouseButtonLeft,
				Action: tea.MouseActionPress,
			}

			result, _ := m.handleMouse(mouseMsg)
			if result.overlay != tt.overlay {
				t.Errorf("overlay = %d, want %d (click inside should not dismiss)", result.overlay, tt.overlay)
			}
		})
	}
}

func TestHandleMouse_OverlayClickOutsideDoesNotFocusPane(t *testing.T) {
	m, _, _ := newTestModel()
	m.focused = PaneFileList
	m.overlay = OverlayHelp
	m.initHelpViewport()

	rects := m.layoutGeometry()

	// Click where the diff pane would be, but outside the overlay
	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      0, // top edge, likely outside centered overlay
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Overlay should dismiss but focus should NOT change to diff pane
	if result.overlay != OverlayNone {
		t.Error("overlay should be dismissed")
	}
	if result.focused != PaneFileList {
		t.Errorf("focus changed to %d, should remain PaneFileList (click should only dismiss)", result.focused)
	}
}

func TestHandleMouse_OverlayMiddleRightClickIgnored(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayHelp
	m.initHelpViewport()

	for _, btn := range []tea.MouseButton{tea.MouseButtonMiddle, tea.MouseButtonRight} {
		mouseMsg := tea.MouseMsg{
			X:      0,
			Y:      0,
			Button: btn,
			Action: tea.MouseActionPress,
		}

		result, cmd := m.handleMouse(mouseMsg)
		if result.overlay != OverlayHelp {
			t.Errorf("button %d: overlay dismissed, should remain", btn)
		}
		if cmd != nil {
			t.Errorf("button %d: expected nil cmd", btn)
		}
	}
}

func TestHandleMouse_OverlayCommitDismissBlursInput(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayCommit
	m.commitInput.Focus()

	// Click outside (top-left corner)
	mouseMsg := tea.MouseMsg{
		X:      0,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.overlay != OverlayNone {
		t.Error("commit overlay should be dismissed")
	}
	if result.commitInput.Focused() {
		t.Error("commitInput should be blurred after dismiss")
	}
}

func TestHandleMouse_HelpOverlayScroll(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayHelp
	m.initHelpViewport()

	rect, _ := m.overlayRect()

	// Scroll down inside the help overlay
	mouseMsg := tea.MouseMsg{
		X:      rect.X + rect.W/2,
		Y:      rect.Y + rect.H/2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.overlay != OverlayHelp {
		t.Error("overlay should remain open during scroll")
	}
	if result.helpViewport.YOffset != 1 {
		t.Errorf("helpViewport.YOffset = %d, want 1", result.helpViewport.YOffset)
	}

	// Scroll up
	m = result
	mouseMsg.Button = tea.MouseButtonWheelUp
	result, _ = m.handleMouse(mouseMsg)
	if result.helpViewport.YOffset != 0 {
		t.Errorf("helpViewport.YOffset = %d, want 0 after scroll up", result.helpViewport.YOffset)
	}
}

func TestHandleMouse_HelpOverlayScrollOutsideIgnored(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayHelp
	m.initHelpViewport()

	// Scroll outside the overlay
	mouseMsg := tea.MouseMsg{
		X:      0,
		Y:      0,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Should not dismiss and should not scroll
	if result.overlay != OverlayHelp {
		t.Error("overlay should remain open")
	}
	if result.helpViewport.YOffset != 0 {
		t.Error("scroll outside overlay should not scroll viewport")
	}
}

func TestHandleMouse_AddFileOverlayScrollMoveCursor(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayAddFile
	w := min(100, max(40, m.width*80/100))
	h := min(30, max(10, m.height*70/100))
	m.addFile = NewAddFileModel([]string{"file1", "file2", "file3"}, w-4, h)

	rect, _ := m.overlayRect()

	// Scroll down inside overlay
	mouseMsg := tea.MouseMsg{
		X:      rect.X + rect.W/2,
		Y:      rect.Y + rect.H/2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.addFile.cursor != 1 {
		t.Errorf("addFile cursor = %d, want 1 after scroll down", result.addFile.cursor)
	}

	// Scroll up
	m = result
	mouseMsg.Button = tea.MouseButtonWheelUp
	result, _ = m.handleMouse(mouseMsg)
	if result.addFile.cursor != 0 {
		t.Errorf("addFile cursor = %d, want 0 after scroll up", result.addFile.cursor)
	}
}

func TestHandleMouse_AddFileOverlayClickTogglesSelection(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayAddFile
	w := min(100, max(40, m.width*80/100))
	h := min(30, max(10, m.height*70/100))
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	m.addFile = NewAddFileModel(files, w-4, h)

	rect, _ := m.overlayRect()

	// Click on a file row inside the overlay.
	// Overlay chrome: border(1) + padding(1) = 2 rows from top
	// Content: title(1) + status(1) = 2 rows header
	// First file row at rect.Y + 2 + 2 = rect.Y + 4
	mouseMsg := tea.MouseMsg{
		X:      rect.X + 5,
		Y:      rect.Y + 4, // first file row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.overlay != OverlayAddFile {
		t.Error("overlay should remain open after click-to-toggle")
	}
	if result.addFile.cursor != 0 {
		t.Errorf("addFile cursor = %d, want 0", result.addFile.cursor)
	}
	if !result.addFile.selected["file1.txt"] {
		t.Error("file1.txt should be selected after click")
	}

	// Click again to deselect
	m = result
	result, _ = m.handleMouse(mouseMsg)
	if result.addFile.selected["file1.txt"] {
		t.Error("file1.txt should be deselected after second click")
	}
}

func TestHandleMouse_AddFileOverlayClickSecondRow(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayAddFile
	w := min(100, max(40, m.width*80/100))
	h := min(30, max(10, m.height*70/100))
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	m.addFile = NewAddFileModel(files, w-4, h)

	rect, _ := m.overlayRect()

	// Click on the second file row
	mouseMsg := tea.MouseMsg{
		X:      rect.X + 5,
		Y:      rect.Y + 5, // second file row (4 + 1)
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.addFile.cursor != 1 {
		t.Errorf("addFile cursor = %d, want 1", result.addFile.cursor)
	}
	if !result.addFile.selected["file2.txt"] {
		t.Error("file2.txt should be selected after click")
	}
}

func TestHandleMouse_AddFileOverlayClickOutOfBounds(t *testing.T) {
	m, _, _ := newTestModel()
	m.overlay = OverlayAddFile
	w := min(100, max(40, m.width*80/100))
	h := min(30, max(10, m.height*70/100))
	m.addFile = NewAddFileModel([]string{"file1.txt"}, w-4, h)

	rect, _ := m.overlayRect()

	// Click on a row beyond the file list (well below the single file)
	mouseMsg := tea.MouseMsg{
		X:      rect.X + 5,
		Y:      rect.Y + rect.H - 3, // near bottom, past file rows
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Should not crash, overlay should remain
	if result.overlay != OverlayAddFile {
		t.Error("overlay should remain open")
	}
	// Selection should not change
	if result.addFile.SelectionCount() != 0 {
		t.Error("no selection should occur for out-of-bounds click")
	}
}

func TestHandleMouse_OverlayScrollOnNonScrollableOverlay(t *testing.T) {
	// Scroll inside a confirm overlay — should be a no-op
	m, _, _ := newTestModel()
	m.overlay = OverlayConfirmApplyAll

	rect, ok := m.overlayRect()
	if !ok {
		t.Fatal("overlayRect should return true")
	}

	mouseMsg := tea.MouseMsg{
		X:      rect.X + rect.W/2,
		Y:      rect.Y + rect.H/2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.overlay != OverlayConfirmApplyAll {
		t.Error("confirm overlay should remain open during scroll")
	}
	if cmd != nil {
		t.Error("expected nil cmd for scroll in confirm overlay")
	}
}

// --- Phase 6: Narrow layout mouse support ---

func TestLayoutGeometry_NarrowLayout(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.updateDimensions()

	rects := m.layoutGeometry()
	contentHeight := m.height - 2

	if !rects.Narrow {
		t.Error("expected narrow layout")
	}

	// All panes span full width
	for _, r := range []Rect{rects.FileList, rects.GitStatus, rects.Status, rects.Diff} {
		if r.X != 0 {
			t.Errorf("pane X = %d, want 0", r.X)
		}
		if r.W != m.width {
			t.Errorf("pane W = %d, want %d", r.W, m.width)
		}
	}

	// FileList is expanded, GitStatus and Status are collapsed
	if rects.FileList.H <= collapsedHeight {
		t.Errorf("FileList should be expanded, got H=%d", rects.FileList.H)
	}
	if rects.GitStatus.H != collapsedHeight {
		t.Errorf("GitStatus should be collapsed, got H=%d", rects.GitStatus.H)
	}
	if rects.Status.H != collapsedHeight {
		t.Errorf("Status should be collapsed, got H=%d", rects.Status.H)
	}
	if rects.Diff.H <= collapsedHeight {
		t.Errorf("Diff should be expanded, got H=%d", rects.Diff.H)
	}

	// Heights sum to contentHeight
	sum := rects.FileList.H + rects.GitStatus.H + rects.Status.H + rects.Diff.H
	if sum != contentHeight {
		t.Errorf("height sum = %d, want %d", sum, contentHeight)
	}
}

func TestHitTest_NarrowCollapsedBars(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click on the collapsed GitStatus bar
	target := hitTest(5, rects.GitStatus.Y, rects)
	if target.IsMiss {
		t.Error("click on collapsed GitStatus bar should not be a miss")
	}
	if target.Pane != PaneGitStatus {
		t.Errorf("pane = %d, want PaneGitStatus", target.Pane)
	}
	if target.RowIndex != -1 {
		t.Errorf("RowIndex = %d, want -1 for collapsed bar", target.RowIndex)
	}

	// Click on the collapsed Status bar
	target = hitTest(5, rects.Status.Y, rects)
	if target.IsMiss {
		t.Error("click on collapsed Status bar should not be a miss")
	}
	if target.Pane != PaneStatus {
		t.Errorf("pane = %d, want PaneStatus", target.Pane)
	}
}

func TestHandleMouse_NarrowClickCollapsedBarExpandsPane(t *testing.T) {
	tests := []struct {
		name       string
		startFocus PaneID
		clickPane  PaneID
	}{
		{"click collapsed GitStatus", PaneFileList, PaneGitStatus},
		{"click collapsed Status", PaneFileList, PaneStatus},
		{"click collapsed FileList", PaneGitStatus, PaneFileList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _, _ := newTestModel()
			m.width = 60
			m.height = 40
			m.focused = tt.startFocus
			m.updateDimensions()

			rects := m.layoutGeometry()

			// Find the collapsed bar's Y coordinate
			var barY int
			switch tt.clickPane {
			case PaneFileList:
				barY = rects.FileList.Y
			case PaneGitStatus:
				barY = rects.GitStatus.Y
			case PaneStatus:
				barY = rects.Status.Y
			}

			mouseMsg := tea.MouseMsg{
				X:      5,
				Y:      barY,
				Button: tea.MouseButtonLeft,
				Action: tea.MouseActionPress,
			}

			result, _ := m.handleMouse(mouseMsg)
			if result.focused != tt.clickPane {
				t.Errorf("focused = %d, want %d", result.focused, tt.clickPane)
			}

			// After focus change, the clicked pane should be expanded
			newRects := result.layoutGeometry()
			var newH int
			switch tt.clickPane {
			case PaneFileList:
				newH = newRects.FileList.H
			case PaneGitStatus:
				newH = newRects.GitStatus.H
			case PaneStatus:
				newH = newRects.Status.H
			}
			if newH <= collapsedHeight {
				t.Errorf("clicked pane should be expanded after click, got H=%d", newH)
			}
		})
	}
}

func TestHandleMouse_NarrowScrollCollapsedBarNoop(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: 'M'},
		{Path: ".gitconfig"},
		{Path: ".vimrc"},
	})
	m.gitStatus.SetEntries([]GitStatusEntry{
		{XY: "M ", Path: "file1.go"},
		{XY: "??", Path: "file2.go"},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()
	origGitCursor := m.gitStatus.cursor

	// Scroll over collapsed GitStatus bar
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.GitStatus.Y,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.gitStatus.cursor != origGitCursor {
		t.Errorf("cursor changed from %d to %d, want no change", origGitCursor, result.gitStatus.cursor)
	}
	if result.focused != PaneFileList {
		t.Error("scroll over collapsed bar should not change focus")
	}
	if cmd != nil {
		t.Error("expected nil cmd for scroll over collapsed bar")
	}

	// Also test scroll up over collapsed Status bar
	mouseMsg.Y = rects.Status.Y
	mouseMsg.Button = tea.MouseButtonWheelUp
	result, cmd = m.handleMouse(mouseMsg)
	if result.focused != PaneFileList {
		t.Error("scroll over collapsed Status bar should not change focus")
	}
	if cmd != nil {
		t.Error("expected nil cmd for scroll over collapsed Status bar")
	}
}

func TestHandleMouse_NarrowClickToSelectExpandedPane(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: 'M'},
		{Path: ".gitconfig"},
		{Path: ".vimrc"},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Click second row in expanded FileList
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 1 + 1, // border + 1 row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, cmd := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1", result.fileList.cursor)
	}
	if result.focused != PaneFileList {
		t.Errorf("focused = %d, want PaneFileList", result.focused)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for diff fetch")
	}
}

func TestHandleMouse_NarrowScrollExpandedPane(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.fileList.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: 'M'},
		{Path: ".gitconfig"},
		{Path: ".vimrc"},
	})
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Scroll down in the expanded FileList
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.FileList.Y + 2, // inside the pane
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.fileList.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after scroll down", result.fileList.cursor)
	}
}

func TestHandleMouse_NarrowScrollDiffPane(t *testing.T) {
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.diffView.ready = true
	m.diffView.viewport.SetContent("line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10")
	m.updateDimensions()

	rects := m.layoutGeometry()

	// Scroll-under-cursor: focus on FileList, scroll over diff pane
	mouseMsg := tea.MouseMsg{
		X:      5,
		Y:      rects.Diff.Y + 2,
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	// Focus should NOT change
	if result.focused != PaneFileList {
		t.Error("scroll-under-cursor should not change focus")
	}
}

func TestHandleMouse_NarrowResizeThenClick(t *testing.T) {
	// Start narrow, resize to wide, click should use new geometry
	m, _, _ := newTestModel()
	m.width = 60
	m.height = 40
	m.focused = PaneFileList
	m.updateDimensions()

	// Resize to wide
	m.width = 120
	m.updateDimensions()

	rects := m.layoutGeometry()
	if rects.Narrow {
		t.Fatal("expected wide layout after resize")
	}

	// Click in the diff pane (right column in wide layout)
	mouseMsg := tea.MouseMsg{
		X:      rects.Diff.X + 5,
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	result, _ := m.handleMouse(mouseMsg)
	if result.focused != PaneInfo {
		t.Errorf("focused = %d, want PaneInfo after clicking diff pane", result.focused)
	}
}
