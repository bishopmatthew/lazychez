package ui

import (
	"testing"

	"github.com/nickrotondo/lazychez/internal/chezmoi"
)

func TestFileItemIsDirty(t *testing.T) {
	tests := []struct {
		name string
		item FileItem
		want bool
	}{
		{"clean", FileItem{AddCol: ' ', ApplyCol: ' '}, false},
		{"add col set", FileItem{AddCol: 'M', ApplyCol: ' '}, true},
		{"apply col set", FileItem{AddCol: ' ', ApplyCol: 'M'}, true},
		{"both set", FileItem{AddCol: 'M', ApplyCol: 'M'}, true},
		{"heading always false", FileItem{AddCol: 'M', ApplyCol: ' ', IsHeading: true}, false},
		{"dir always false", FileItem{AddCol: 'M', ApplyCol: ' ', IsDir: true}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.item.IsDirty(); got != tt.want {
				t.Errorf("IsDirty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileItemStatusCode(t *testing.T) {
	tests := []struct {
		add, apply rune
		want       string
	}{
		{'M', 'M', "MM"},
		{' ', 'A', " A"},
		{'D', ' ', "D "},
		{' ', ' ', "  "},
	}
	for _, tt := range tests {
		item := FileItem{AddCol: tt.add, ApplyCol: tt.apply}
		if got := item.StatusCode(); got != tt.want {
			t.Errorf("StatusCode() = %q, want %q", got, tt.want)
		}
	}
}

func TestMergeFilesWithStatus(t *testing.T) {
	t.Run("no status entries", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".gitconfig", SourceRelPath: "dot_gitconfig"},
		}
		items := MergeFilesWithStatus(managed, nil)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
		for _, item := range items {
			if item.IsDirty() {
				t.Errorf("item %q should not be dirty", item.Path)
			}
		}
	})

	t.Run("status merges with managed", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		status := []chezmoi.StatusEntry{
			{AddCol: ' ', ApplyCol: 'M', Path: ".zshrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		if len(items) != 1 {
			t.Fatalf("got %d items, want 1", len(items))
		}
		if !items[0].IsDirty() {
			t.Error("item should be dirty")
		}
		if items[0].ApplyCol != 'M' {
			t.Errorf("ApplyCol = %c, want M", items[0].ApplyCol)
		}
	})

	t.Run("orphan status entry", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
		}
		status := []chezmoi.StatusEntry{
			{AddCol: 'M', ApplyCol: 'A', Path: ".bashrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		if len(items) != 2 {
			t.Fatalf("got %d items, want 2", len(items))
		}
		var orphan *FileItem
		for i := range items {
			if items[i].Path == ".bashrc" {
				orphan = &items[i]
			}
		}
		if orphan == nil {
			t.Fatal("orphan .bashrc not found")
		}
		if orphan.AddCol != 'M' || orphan.ApplyCol != 'A' {
			t.Errorf("orphan cols = %c%c, want MA", orphan.AddCol, orphan.ApplyCol)
		}
	})

	t.Run("dirty files sort before clean", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".vimrc", SourceRelPath: "dot_vimrc"},
		}
		status := []chezmoi.StatusEntry{
			{AddCol: ' ', ApplyCol: 'M', Path: ".vimrc"},
		}
		items := MergeFilesWithStatus(managed, status)
		if len(items) != 3 {
			t.Fatalf("got %d items, want 3", len(items))
		}
		// Dirty first, then clean alphabetical
		if items[0].Path != ".vimrc" || !items[0].IsDirty() {
			t.Errorf("items[0] = %q dirty=%v, want .vimrc dirty", items[0].Path, items[0].IsDirty())
		}
		if items[1].Path != ".bashrc" {
			t.Errorf("items[1] = %q, want .bashrc", items[1].Path)
		}
		if items[2].Path != ".zshrc" {
			t.Errorf("items[2] = %q, want .zshrc", items[2].Path)
		}
	})

	t.Run("dirty files sort by status code then alphabetically", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
		}
		status := []chezmoi.StatusEntry{
			{AddCol: 'M', ApplyCol: 'M', Path: ".zshrc"},  // "MM"
			{AddCol: ' ', ApplyCol: 'M', Path: ".bashrc"},  // " M"
		}
		items := MergeFilesWithStatus(managed, status)
		// " M" < "MM" lexicographically (space < 'M')
		if items[0].Path != ".bashrc" {
			t.Errorf("items[0] = %q, want .bashrc (status ' M' < 'MM')", items[0].Path)
		}
		if items[1].Path != ".zshrc" {
			t.Errorf("items[1] = %q, want .zshrc", items[1].Path)
		}
	})

	t.Run("alphabetical within same status code", func(t *testing.T) {
		managed := []chezmoi.ManagedFile{
			{Path: ".zshrc", SourceRelPath: "dot_zshrc"},
			{Path: ".bashrc", SourceRelPath: "dot_bashrc"},
			{Path: ".config", SourceRelPath: "dot_config"},
		}
		items := MergeFilesWithStatus(managed, nil)
		// All clean, should be sorted alphabetically
		if items[0].Path != ".bashrc" {
			t.Errorf("items[0].Path = %q, want .bashrc", items[0].Path)
		}
		if items[1].Path != ".config" {
			t.Errorf("items[1].Path = %q, want .config", items[1].Path)
		}
		if items[2].Path != ".zshrc" {
			t.Errorf("items[2].Path = %q, want .zshrc", items[2].Path)
		}
	})

	t.Run("empty inputs", func(t *testing.T) {
		items := MergeFilesWithStatus(nil, nil)
		if len(items) != 0 {
			t.Errorf("got %d items, want 0", len(items))
		}
	})
}

func TestInsertDivider(t *testing.T) {
	t.Run("no dirty files produces no divider", func(t *testing.T) {
		files := []FileItem{
			{Path: ".zshrc", AddCol: ' ', ApplyCol: ' '},
			{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
		}
		result := insertDivider(files)
		for _, f := range result {
			if f.IsHeading {
				t.Error("unexpected divider in result with no dirty files")
			}
		}
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
	})

	t.Run("all dirty files produces no divider", func(t *testing.T) {
		files := []FileItem{
			{Path: ".zshrc", AddCol: 'M', ApplyCol: ' '},
			{Path: ".vimrc", AddCol: ' ', ApplyCol: 'M'},
		}
		result := insertDivider(files)
		for _, f := range result {
			if f.IsHeading {
				t.Error("unexpected divider when all files are dirty")
			}
		}
	})

	t.Run("mixed dirty and clean gets divider", func(t *testing.T) {
		files := []FileItem{
			{Path: ".zshrc", AddCol: 'M', ApplyCol: 'M'},
			{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
		}
		result := insertDivider(files)
		if len(result) != 3 {
			t.Fatalf("len = %d, want 3", len(result))
		}
		if result[0].Path != ".zshrc" {
			t.Errorf("result[0].Path = %q, want .zshrc", result[0].Path)
		}
		if !result[1].IsHeading {
			t.Error("result[1] should be a divider")
		}
		if result[2].Path != ".vimrc" {
			t.Errorf("result[2].Path = %q, want .vimrc", result[2].Path)
		}
	})
}

func TestFileListModel_Navigation(t *testing.T) {
	makeModel := func() FileListModel {
		m := NewFileListModel()
		m.SetDimensions(80, 20)
		// Simulate files with a divider: [file0, divider, file1, file2]
		m.files = []FileItem{
			{Path: ".bashrc", AddCol: 'M', ApplyCol: ' '},
			{IsHeading: true},
			{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
			{Path: ".zshrc", AddCol: ' ', ApplyCol: ' '},
		}
		m.cursor = 0 // start on first file
		return m
	}

	t.Run("MoveDown skips divider", func(t *testing.T) {
		m := makeModel()
		m.MoveDown() // from .bashrc (0), should skip divider (1), land on .vimrc (2)
		if m.cursor != 2 {
			t.Errorf("cursor = %d, want 2", m.cursor)
		}
	})

	t.Run("MoveUp skips divider", func(t *testing.T) {
		m := makeModel()
		m.cursor = 2 // .vimrc
		m.MoveUp()   // should skip divider (1), land on .bashrc (0)
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("MoveDown at bottom is no-op", func(t *testing.T) {
		m := makeModel()
		m.cursor = 3 // .zshrc (last)
		m.MoveDown()
		if m.cursor != 3 {
			t.Errorf("cursor = %d, want 3", m.cursor)
		}
	})

	t.Run("MoveUp at top is no-op", func(t *testing.T) {
		m := makeModel()
		m.cursor = 0 // .bashrc (first file)
		m.MoveUp()
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("GoToTop lands on first file not divider", func(t *testing.T) {
		m := makeModel()
		m.cursor = 3
		m.GoToTop()
		if m.cursor != 0 {
			t.Errorf("cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("GoToBottom lands on last file", func(t *testing.T) {
		m := makeModel()
		m.cursor = 0
		m.GoToBottom()
		if m.cursor != 3 {
			t.Errorf("cursor = %d, want 3", m.cursor)
		}
	})

	t.Run("empty list navigation", func(t *testing.T) {
		m := NewFileListModel()
		m.MoveDown()
		m.MoveUp()
		m.GoToTop()
		m.GoToBottom()
		// Should not panic
	})
}

func TestFileListModel_Queries(t *testing.T) {
	t.Run("SelectedPath empty list", func(t *testing.T) {
		m := NewFileListModel()
		if got := m.SelectedPath(); got != "" {
			t.Errorf("SelectedPath() = %q, want empty", got)
		}
	})

	t.Run("SelectedPath on divider returns empty", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{IsHeading: true}}
		m.cursor = 0
		if got := m.SelectedPath(); got != "" {
			t.Errorf("SelectedPath() = %q, want empty", got)
		}
	})

	t.Run("SelectedPath on file", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{Path: ".zshrc"}}
		m.cursor = 0
		if got := m.SelectedPath(); got != ".zshrc" {
			t.Errorf("SelectedPath() = %q, want .zshrc", got)
		}
	})

	t.Run("SelectedItem nil on divider", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{IsHeading: true}}
		m.cursor = 0
		if got := m.SelectedItem(); got != nil {
			t.Error("SelectedItem() should be nil for divider")
		}
	})

	t.Run("SelectedItem returns item", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{{Path: ".zshrc", AddCol: 'M', ApplyCol: ' '}}
		m.cursor = 0
		got := m.SelectedItem()
		if got == nil {
			t.Fatal("SelectedItem() = nil, want item")
		}
		if got.Path != ".zshrc" {
			t.Errorf("SelectedItem().Path = %q, want .zshrc", got.Path)
		}
	})

	t.Run("FileCount excludes dividers", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{
			{Path: ".zshrc"},
			{IsHeading: true},
			{Path: ".vimrc"},
		}
		if got := m.FileCount(); got != 2 {
			t.Errorf("FileCount() = %d, want 2", got)
		}
	})

	t.Run("DirtyCount counts only dirty files", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{
			{Path: ".zshrc", AddCol: 'M', ApplyCol: ' '},
			{IsHeading: true},
			{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
		}
		if got := m.DirtyCount(); got != 1 {
			t.Errorf("DirtyCount() = %d, want 1", got)
		}
	})

	t.Run("ContentLines includes dividers", func(t *testing.T) {
		m := NewFileListModel()
		m.files = []FileItem{
			{Path: ".zshrc"},
			{IsHeading: true},
			{Path: ".vimrc"},
		}
		if got := m.ContentLines(); got != 3 {
			t.Errorf("ContentLines() = %d, want 3", got)
		}
	})
}

func TestFileListModel_SetFiles(t *testing.T) {
	t.Run("clamps cursor when list shrinks", func(t *testing.T) {
		m := NewFileListModel()
		m.SetDimensions(80, 20)
		m.files = make([]FileItem, 10)
		m.cursor = 8

		// Set a smaller list (no dirty files so no divider)
		smallList := []FileItem{
			{Path: ".a", AddCol: ' ', ApplyCol: ' '},
			{Path: ".b", AddCol: ' ', ApplyCol: ' '},
		}
		m.SetFiles(smallList)
		if m.cursor >= len(m.files) {
			t.Errorf("cursor = %d, should be < len(files)=%d", m.cursor, len(m.files))
		}
	})

	t.Run("snaps cursor off divider", func(t *testing.T) {
		m := NewFileListModel()
		m.SetDimensions(80, 20)

		// Files that will produce a divider — cursor might land on one after clamp
		files := []FileItem{
			{Path: ".zshrc", AddCol: ' ', ApplyCol: 'M'},
			{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
		}
		m.SetFiles(files)
		// After SetFiles, cursor should be on a file, not a divider
		if m.cursor < len(m.files) && m.files[m.cursor].IsHeading {
			t.Errorf("cursor at %d is on a divider", m.cursor)
		}
	})
}

// makeFilterModel creates a FileListModel pre-loaded with files for filter tests.
func makeFilterModel() FileListModel {
	m := NewFileListModel()
	m.SetDimensions(80, 20)
	m.SetFiles([]FileItem{
		{Path: ".bashrc", SourceRelPath: "dot_bashrc", AddCol: ' ', ApplyCol: 'M'},
		{Path: ".zshrc", SourceRelPath: "dot_zshrc", AddCol: 'M', ApplyCol: ' '},
		{Path: ".vimrc", SourceRelPath: "dot_vimrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", SourceRelPath: "dot_gitconfig", AddCol: ' ', ApplyCol: ' '},
	})
	return m
}

func TestFilter_FuzzyMatching(t *testing.T) {
	t.Run("exact substring match", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("zsh")
		m.applyFilter()

		if m.FileCount() != 1 {
			t.Fatalf("FileCount() = %d, want 1", m.FileCount())
		}
		if m.SelectedPath() != ".zshrc" {
			t.Errorf("SelectedPath() = %q, want .zshrc", m.SelectedPath())
		}
	})

	t.Run("fuzzy match across characters", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("brc")
		m.applyFilter()

		if m.FileCount() < 1 {
			t.Fatalf("FileCount() = %d, want >= 1", m.FileCount())
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("ZSH")
		m.applyFilter()

		if m.FileCount() != 1 {
			t.Fatalf("FileCount() = %d, want 1", m.FileCount())
		}
		if m.SelectedPath() != ".zshrc" {
			t.Errorf("SelectedPath() = %q, want .zshrc", m.SelectedPath())
		}
	})

	t.Run("multiple matches", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("rc")
		m.applyFilter()

		// .bashrc, .zshrc, .vimrc all contain "rc"
		if m.FileCount() < 3 {
			t.Fatalf("FileCount() = %d, want >= 3", m.FileCount())
		}
	})
}

func TestFilter_EmptyQuery(t *testing.T) {
	m := makeFilterModel()
	totalBefore := m.FileCount()

	m.StartFilter()
	m.filterInput.SetValue("")
	m.applyFilter()

	if m.FileCount() != totalBefore {
		t.Errorf("FileCount() = %d, want %d (all items)", m.FileCount(), totalBefore)
	}
}

func TestFilter_NoMatches(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("zzzznotafile")
	m.applyFilter()

	if m.FileCount() != 0 {
		t.Errorf("FileCount() = %d, want 0", m.FileCount())
	}
	if m.FilterHasMatches() {
		t.Error("FilterHasMatches() = true, want false")
	}
}

func TestFilter_NavigationLocked(t *testing.T) {
	t.Run("j/k navigate only filtered items", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("sh")
		m.applyFilter()

		if m.FileCount() != 2 {
			t.Fatalf("FileCount() = %d, want 2", m.FileCount())
		}

		locked := m.LockFilter()
		if !locked {
			t.Fatal("LockFilter() returned false")
		}

		// Cursor should be on first match
		first := m.SelectedPath()
		m.MoveDown()
		second := m.SelectedPath()

		if first == second {
			t.Error("MoveDown did not change selected path")
		}

		// MoveDown again should be no-op (only 2 items)
		m.MoveDown()
		if m.SelectedPath() != second {
			t.Errorf("MoveDown past last item changed path to %q", m.SelectedPath())
		}

		// MoveUp back to first
		m.MoveUp()
		if m.SelectedPath() != first {
			t.Errorf("MoveUp should return to %q, got %q", first, m.SelectedPath())
		}
	})
}

func TestFilter_EscDuringTyping(t *testing.T) {
	m := makeFilterModel()
	totalBefore := m.FileCount()
	m.cursor = 3 // put cursor on a specific item
	cursorBefore := m.cursor

	m.StartFilter()
	m.filterInput.SetValue("zsh")
	m.applyFilter()

	if m.FileCount() == totalBefore {
		t.Fatal("filter should have reduced visible items")
	}

	m.CancelFilter()

	if m.filterMode != FilterInactive {
		t.Errorf("filterMode = %d, want FilterInactive", m.filterMode)
	}
	if m.FileCount() != totalBefore {
		t.Errorf("FileCount() = %d, want %d (restored)", m.FileCount(), totalBefore)
	}
	if m.cursor != cursorBefore {
		t.Errorf("cursor = %d, want %d (restored)", m.cursor, cursorBefore)
	}
}

func TestFilter_EnterLocks(t *testing.T) {
	t.Run("lock with matches", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("vim")
		m.applyFilter()

		locked := m.LockFilter()
		if !locked {
			t.Fatal("LockFilter() returned false")
		}
		if m.filterMode != FilterLocked {
			t.Errorf("filterMode = %d, want FilterLocked", m.filterMode)
		}
		if !m.IsFilterLocked() {
			t.Error("IsFilterLocked() = false")
		}
		// Cursor should be on first match
		if m.SelectedPath() != ".vimrc" {
			t.Errorf("SelectedPath() = %q, want .vimrc", m.SelectedPath())
		}
	})

	t.Run("lock with no matches is no-op", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("zzzznotafile")
		m.applyFilter()

		locked := m.LockFilter()
		if locked {
			t.Error("LockFilter() should return false when no matches")
		}
		if m.filterMode != FilterTyping {
			t.Errorf("filterMode = %d, want FilterTyping (unchanged)", m.filterMode)
		}
	})
}

func TestFilter_EscDuringLocked(t *testing.T) {
	m := makeFilterModel()
	totalBefore := m.FileCount()

	m.StartFilter()
	m.filterInput.SetValue("vim")
	m.applyFilter()
	m.LockFilter()

	if m.IsFilterLocked() != true {
		t.Fatal("expected locked filter mode")
	}

	m.CancelFilter()

	if m.filterMode != FilterInactive {
		t.Errorf("filterMode = %d, want FilterInactive", m.filterMode)
	}
	if m.FileCount() != totalBefore {
		t.Errorf("FileCount() = %d, want %d", m.FileCount(), totalBefore)
	}
}

func TestFilter_PreservedOnDataRefresh(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("vim")
	m.applyFilter()
	m.LockFilter()

	// Simulate data refresh by calling SetFiles — filter should be reapplied
	m.SetFiles([]FileItem{
		{Path: ".bashrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".vimrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".newfile", AddCol: ' ', ApplyCol: ' '},
	})

	if m.filterMode != FilterLocked {
		t.Errorf("filterMode = %d, want FilterLocked after SetFiles", m.filterMode)
	}
	// Only .vimrc matches "vim"
	if m.FileCount() != 1 {
		t.Errorf("FileCount() = %d, want 1", m.FileCount())
	}
}

func TestFilter_DividerInFilteredList(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	// Match only clean files (no dirty)
	m.filterInput.SetValue("vim")
	m.applyFilter()

	// Only .vimrc should match — it's clean
	if m.FileCount() != 1 {
		t.Fatalf("FileCount() = %d, want 1", m.FileCount())
	}

	// No divider should appear since all matches are clean
	for _, f := range m.files {
		if f.IsHeading {
			t.Error("unexpected divider in filtered list with only clean files")
		}
	}
}

func TestFilter_FileOperationsOnFilteredItems(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("bashrc")
	m.applyFilter()
	m.LockFilter()

	// SelectedItem should return the filtered item, not a different one
	item := m.SelectedItem()
	if item == nil {
		t.Fatal("SelectedItem() = nil")
	}
	if item.Path != ".bashrc" {
		t.Errorf("SelectedItem().Path = %q, want .bashrc", item.Path)
	}
	if !item.IsDirty() {
		t.Error("SelectedItem() should be dirty")
	}
}

func TestFilter_ModeTransitions(t *testing.T) {
	t.Run("inactive → typing → locked → inactive", func(t *testing.T) {
		m := makeFilterModel()

		if m.filterMode != FilterInactive {
			t.Fatalf("initial filterMode = %d, want FilterInactive", m.filterMode)
		}

		m.StartFilter()
		if m.filterMode != FilterTyping {
			t.Fatalf("after StartFilter: filterMode = %d, want FilterTyping", m.filterMode)
		}
		if !m.IsFiltering() {
			t.Error("IsFiltering() = false after StartFilter")
		}

		m.filterInput.SetValue("vim")
		m.applyFilter()
		m.LockFilter()
		if m.filterMode != FilterLocked {
			t.Fatalf("after LockFilter: filterMode = %d, want FilterLocked", m.filterMode)
		}

		m.CancelFilter()
		if m.filterMode != FilterInactive {
			t.Fatalf("after CancelFilter: filterMode = %d, want FilterInactive", m.filterMode)
		}
	})

	t.Run("inactive → typing → cancel → inactive", func(t *testing.T) {
		m := makeFilterModel()
		m.StartFilter()
		m.filterInput.SetValue("zsh")
		m.applyFilter()

		m.CancelFilter()
		if m.filterMode != FilterInactive {
			t.Errorf("filterMode = %d, want FilterInactive", m.filterMode)
		}
		if m.IsFiltering() {
			t.Error("IsFiltering() = true after cancel")
		}
		if m.IsFilterLocked() {
			t.Error("IsFilterLocked() = true after cancel")
		}
	})
}

func TestFilter_CursorPosition(t *testing.T) {
	m := makeFilterModel()
	m.StartFilter()
	m.filterInput.SetValue("sh")
	m.applyFilter()
	m.LockFilter()

	current, total := m.CursorPosition()
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if current != 1 {
		t.Errorf("current = %d, want 1 (first match)", current)
	}

	m.MoveDown()
	current, total = m.CursorPosition()
	if current != 2 {
		t.Errorf("after MoveDown: current = %d, want 2", current)
	}
	if total != 2 {
		t.Errorf("after MoveDown: total = %d, want 2", total)
	}
}

// --- Search tests ---

// makeSearchModel creates a FileListModel pre-loaded with files for search tests.
// Uses a flat list (no tree) for predictable index positions.
func makeSearchModel() FileListModel {
	m := NewFileListModel()
	m.SetDimensions(80, 20)
	m.SetFiles([]FileItem{
		{Path: ".bashrc", TreeName: ".bashrc", AddCol: ' ', ApplyCol: 'M'},
		{Path: ".zshrc", TreeName: ".zshrc", AddCol: 'M', ApplyCol: ' '},
		{Path: ".vimrc", TreeName: ".vimrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", TreeName: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
	})
	return m
}

func TestSearch_IncrementalJump(t *testing.T) {
	m := makeSearchModel()
	totalBefore := len(m.files)

	m.StartSearch()
	m.searchInput.SetValue("vim")
	m.computeSearchMatches()
	m.jumpToNextSearchMatch()

	// All files should still be visible
	if len(m.files) != totalBefore {
		t.Errorf("len(files) = %d, want %d (all files visible)", len(m.files), totalBefore)
	}

	// Cursor should have jumped to .vimrc
	if m.SelectedPath() != ".vimrc" {
		t.Errorf("SelectedPath() = %q, want .vimrc", m.SelectedPath())
	}

	if len(m.searchMatches) != 1 {
		t.Errorf("searchMatches = %d, want 1", len(m.searchMatches))
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("ZSH")
	m.computeSearchMatches()

	if len(m.searchMatches) != 1 {
		t.Fatalf("searchMatches = %d, want 1", len(m.searchMatches))
	}

	m.jumpToNextSearchMatch()
	if m.SelectedPath() != ".zshrc" {
		t.Errorf("SelectedPath() = %q, want .zshrc", m.SelectedPath())
	}
}

func TestSearch_NoMatches(t *testing.T) {
	m := makeSearchModel()
	m.SetCursor(2) // start on a specific file
	savedPos := m.cursor

	m.StartSearch()
	m.searchInput.SetValue("nonexistent")
	m.computeSearchMatches()

	if len(m.searchMatches) != 0 {
		t.Errorf("searchMatches = %d, want 0", len(m.searchMatches))
	}

	// jumpToNextSearchMatch should return false and not move cursor
	moved := m.jumpToNextSearchMatch()
	if moved {
		t.Error("jumpToNextSearchMatch() = true, want false")
	}

	// Cancel should restore original position
	m.CancelSearch()
	if m.cursor != savedPos {
		t.Errorf("cursor = %d, want %d (restored)", m.cursor, savedPos)
	}
}

func TestSearch_NextPrevMatch(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("rc") // matches .bashrc, .zshrc, .vimrc
	m.computeSearchMatches()

	if len(m.searchMatches) < 3 {
		t.Fatalf("searchMatches = %d, want >= 3", len(m.searchMatches))
	}

	m.ConfirmSearch()
	if m.searchMode != SearchActive {
		t.Fatalf("searchMode = %d, want SearchActive", m.searchMode)
	}

	// Start at first match
	m.cursor = m.searchMatches[0]
	first := m.cursor

	// Next should go to second match
	m.NextSearchMatch()
	second := m.cursor
	if second <= first {
		t.Errorf("NextSearchMatch: cursor %d should be > %d", second, first)
	}

	// Prev should go back to first
	m.PrevSearchMatch()
	if m.cursor != first {
		t.Errorf("PrevSearchMatch: cursor = %d, want %d", m.cursor, first)
	}
}

func TestSearch_WrapAround(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("rc") // matches .bashrc, .zshrc, .vimrc
	m.computeSearchMatches()
	m.ConfirmSearch()

	if len(m.searchMatches) < 2 {
		t.Fatalf("need >= 2 matches, got %d", len(m.searchMatches))
	}

	lastMatch := m.searchMatches[len(m.searchMatches)-1]
	firstMatch := m.searchMatches[0]

	// Go to last match, then next should wrap to first
	m.cursor = lastMatch
	m.NextSearchMatch()
	if m.cursor != firstMatch {
		t.Errorf("NextSearchMatch wrap: cursor = %d, want %d", m.cursor, firstMatch)
	}

	// Go to first match, then prev should wrap to last
	m.cursor = firstMatch
	m.PrevSearchMatch()
	if m.cursor != lastMatch {
		t.Errorf("PrevSearchMatch wrap: cursor = %d, want %d", m.cursor, lastMatch)
	}
}

func TestSearch_EscDuringTyping(t *testing.T) {
	m := makeSearchModel()
	// Move to last file so searching for "bash" jumps away
	m.GoToBottom()
	savedPos := m.cursor

	m.StartSearch()
	if m.searchMode != SearchTyping {
		t.Fatalf("searchMode = %d, want SearchTyping", m.searchMode)
	}

	m.searchInput.SetValue("gitconfig")
	m.computeSearchMatches()
	if len(m.searchMatches) == 0 {
		t.Fatal("expected at least 1 match for 'gitconfig'")
	}
	m.jumpToNextSearchMatch()

	// Cancel restores cursor
	m.CancelSearch()
	if m.searchMode != SearchInactive {
		t.Errorf("searchMode = %d, want SearchInactive", m.searchMode)
	}
	if m.cursor != savedPos {
		t.Errorf("cursor = %d, want %d (restored)", m.cursor, savedPos)
	}
}

func TestSearch_EscDuringActive(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("vim")
	m.computeSearchMatches()
	m.jumpToNextSearchMatch()
	m.ConfirmSearch()

	cursorAfterConfirm := m.cursor

	// Clear search — cursor should stay where it is (vim behavior)
	m.ClearSearch()
	if m.searchMode != SearchInactive {
		t.Errorf("searchMode = %d, want SearchInactive", m.searchMode)
	}
	if m.cursor != cursorAfterConfirm {
		t.Errorf("cursor = %d, want %d (stays put)", m.cursor, cursorAfterConfirm)
	}
	if len(m.searchMatches) != 0 {
		t.Errorf("searchMatches = %d, want 0 (cleared)", len(m.searchMatches))
	}
}

func TestSearch_ModeTransitions(t *testing.T) {
	m := makeSearchModel()

	// Inactive → Typing
	if m.searchMode != SearchInactive {
		t.Fatalf("initial: searchMode = %d, want SearchInactive", m.searchMode)
	}
	m.StartSearch()
	if m.searchMode != SearchTyping {
		t.Fatalf("after StartSearch: searchMode = %d, want SearchTyping", m.searchMode)
	}
	if !m.IsSearching() {
		t.Error("IsSearching() = false, want true")
	}

	// Typing → Active
	m.searchInput.SetValue("rc")
	m.computeSearchMatches()
	m.jumpToNextSearchMatch()
	m.ConfirmSearch()
	if m.searchMode != SearchActive {
		t.Fatalf("after ConfirmSearch: searchMode = %d, want SearchActive", m.searchMode)
	}
	if !m.IsSearchActive() {
		t.Error("IsSearchActive() = false, want true")
	}
	if m.IsSearching() {
		t.Error("IsSearching() = true during SearchActive, want false")
	}

	// Active → Inactive
	m.ClearSearch()
	if m.searchMode != SearchInactive {
		t.Errorf("after ClearSearch: searchMode = %d, want SearchInactive", m.searchMode)
	}

	// Typing → Inactive (cancel)
	m.StartSearch()
	m.CancelSearch()
	if m.searchMode != SearchInactive {
		t.Errorf("after CancelSearch: searchMode = %d, want SearchInactive", m.searchMode)
	}
}

func TestSearch_HighlightDetection(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("bash")
	m.computeSearchMatches()

	if len(m.searchMatches) != 1 {
		t.Fatalf("searchMatches = %d, want 1", len(m.searchMatches))
	}

	matchIdx := m.searchMatches[0]
	if !m.isSearchMatch(matchIdx) {
		t.Errorf("isSearchMatch(%d) = false, want true", matchIdx)
	}

	// Non-match index should return false
	for i := range m.files {
		if i != matchIdx && m.isSearchMatch(i) {
			t.Errorf("isSearchMatch(%d) = true, want false (not a match)", i)
		}
	}
}

func TestSearch_HighlightInactiveReturns_False(t *testing.T) {
	m := makeSearchModel()
	// SearchInactive — isSearchMatch should always return false
	for i := range m.files {
		if m.isSearchMatch(i) {
			t.Errorf("isSearchMatch(%d) = true during SearchInactive, want false", i)
		}
	}
}

func TestSearch_SurvivesDataRefresh(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("zsh")
	m.computeSearchMatches()

	if len(m.searchMatches) != 1 {
		t.Fatalf("before refresh: searchMatches = %d, want 1", len(m.searchMatches))
	}

	// Refresh with new data that still includes .zshrc
	m.SetFiles([]FileItem{
		{Path: ".zshrc", TreeName: ".zshrc", AddCol: 'M', ApplyCol: ' '},
		{Path: ".newfile", TreeName: ".newfile", AddCol: 'A', ApplyCol: ' '},
	})

	if len(m.searchMatches) != 1 {
		t.Errorf("after refresh: searchMatches = %d, want 1", len(m.searchMatches))
	}
}

func TestSearch_WithActiveFilter(t *testing.T) {
	m := makeFilterModel()

	// Apply filter first to narrow down to "rc" files
	m.StartFilter()
	m.filterInput.SetValue("rc")
	m.applyFilter()
	filteredCount := m.FileCount()
	if filteredCount < 3 {
		t.Fatalf("filter FileCount() = %d, want >= 3", filteredCount)
	}

	// Now search within filtered results
	m.StartSearch()
	m.searchInput.SetValue("vim")
	m.computeSearchMatches()

	if len(m.searchMatches) != 1 {
		t.Errorf("searchMatches within filter = %d, want 1", len(m.searchMatches))
	}

	// File count should still be the filtered count (search doesn't hide files)
	if m.FileCount() != filteredCount {
		t.Errorf("FileCount() = %d, want %d (search doesn't hide)", m.FileCount(), filteredCount)
	}
}

func TestSearch_ConfirmWithNoMatchesStaysTyping(t *testing.T) {
	m := makeSearchModel()
	m.StartSearch()
	m.searchInput.SetValue("nonexistent")
	m.computeSearchMatches()

	m.ConfirmSearch()
	// Should stay in typing mode since there are no matches
	if m.searchMode != SearchTyping {
		t.Errorf("searchMode = %d, want SearchTyping (no matches to confirm)", m.searchMode)
	}
}
