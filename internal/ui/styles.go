package ui

import "github.com/charmbracelet/lipgloss"

func adaptiveColor(light, dark string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: light, Dark: dark}
}

var (
	// Border colors
	ActiveBorderColor   = adaptiveColor("#0F6E8C", "#81CAE4")
	InactiveBorderColor = adaptiveColor("#6B7F87", "#5A7A86")

	// Base text colors
	TextColor  = adaptiveColor("#102129", "#E9F6FB")
	MutedColor = adaptiveColor("#5F7077", "#5A7A86")

	// Semantic colors
	ModifiedColor = adaptiveColor("#8A6400", "#E4CD81")
	AddedColor    = adaptiveColor("#1E7A1E", "#98E481")
	DeletedColor  = adaptiveColor("#B42318", "#E48281")
	TitleColor    = adaptiveColor("#0F6E8C", "#81CAE4")
	SelectedBg    = adaptiveColor("#D7ECF4", "#114A5F")
	DirColor      = adaptiveColor("#295D70", "#A2C5D2")
	TemplateColor = adaptiveColor("#006F7A", "#26D6D9")
	SuccessColor  = adaptiveColor("#1E7A1E", "#98E481")
	ErrorColor    = adaptiveColor("#B42318", "#E48281")

	// Search highlight
	SearchMatchBg = adaptiveColor("#FFF3CD", "#3D3200")

	// Diff colors
	DiffAddColor  = adaptiveColor("#1E7A1E", "#98E481")
	DiffDelColor  = adaptiveColor("#B42318", "#E48281")
	DiffHunkColor = adaptiveColor("#6C46C0", "#9C81E4")
	DiffMetaColor = adaptiveColor("#5F7077", "#5A7A86")

	// Overlay colors
	OverlayBackgroundColor = adaptiveColor("#F7FAFB", "#172B32")
	OverlayBackdropColor   = adaptiveColor("#DCE9EE", "#172B32")

	// Pane styles
	ActivePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveBorderColor)

	InactivePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(InactiveBorderColor)

	// Title style (rendered as first line inside pane)
	PaneTitle = lipgloss.NewStyle().
			Foreground(TitleColor).
			Bold(true)

	// File list
	SelectedItem = lipgloss.NewStyle().
			Background(SelectedBg).
			Bold(true)

	NormalItem      = lipgloss.NewStyle().Foreground(TextColor)
	SearchHighlight = lipgloss.NewStyle().Background(SearchMatchBg)

	// Status indicators
	AddedIndicator   = lipgloss.NewStyle().Foreground(AddedColor).SetString("+")
	DeletedIndicator = lipgloss.NewStyle().Foreground(DeletedColor).SetString("−")

	// Diff line styles
	DiffAdd  = lipgloss.NewStyle().Foreground(DiffAddColor)
	DiffDel  = lipgloss.NewStyle().Foreground(DiffDelColor)
	DiffHunk = lipgloss.NewStyle().Foreground(DiffHunkColor)
	DiffMeta = lipgloss.NewStyle().Foreground(DiffMetaColor)

	// Footer
	HelpKey    = lipgloss.NewStyle().Foreground(ActiveBorderColor).Bold(true)
	HelpDesc   = lipgloss.NewStyle().Foreground(MutedColor)
	HelpSep    = lipgloss.NewStyle().Foreground(MutedColor)
	FooterLink = lipgloss.NewStyle().Foreground(TextColor).Underline(true)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextColor)
	StatusBarError = lipgloss.NewStyle().
			Foreground(ErrorColor)
	StatusBarSuccess = lipgloss.NewStyle().
				Foreground(SuccessColor)

	// Overlay (help, commit input, confirm dialogs)
	OverlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ActiveBorderColor).
			Padding(1, 2).
			Background(OverlayBackgroundColor)
)
