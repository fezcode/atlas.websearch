package ui

import "github.com/charmbracelet/lipgloss"

// Phosphor-CRT telemetry palette.
// Amber primary on true-black, cyan numeric readouts, red criticals —
// evocative of 1970s engineering workstation displays.
var (
	ColBG       = lipgloss.Color("#000000")
	ColChrome   = lipgloss.Color("#3A3226") // structural borders
	ColDim      = lipgloss.Color("#7A6A4A") // secondary labels
	ColText     = lipgloss.Color("#D9C79C") // body text
	ColAmber    = lipgloss.Color("#FFB000") // primary phosphor
	ColAmberHot = lipgloss.Color("#FF7A00") // elevated
	ColRed      = lipgloss.Color("#FF3D4A") // critical
	ColCyan     = lipgloss.Color("#7DE3FF") // numeric readouts
	ColGreen    = lipgloss.Color("#84F5A3") // nominal
	ColPaper    = lipgloss.Color("#F5E6D3") // headings
)

// Reusable styles.
var (
	sBorder       = lipgloss.NewStyle().Foreground(ColChrome)
	sLabel        = lipgloss.NewStyle().Foreground(ColDim)
	sText         = lipgloss.NewStyle().Foreground(ColText)
	sValue        = lipgloss.NewStyle().Foreground(ColCyan).Bold(true)
	sPaper        = lipgloss.NewStyle().Foreground(ColPaper).Bold(true)
	sAmber        = lipgloss.NewStyle().Foreground(ColAmber).Bold(true)
	sHot          = lipgloss.NewStyle().Foreground(ColAmberHot).Bold(true)
	sCrit         = lipgloss.NewStyle().Foreground(ColRed).Bold(true)
	sGood         = lipgloss.NewStyle().Foreground(ColGreen).Bold(true)
	sDim          = lipgloss.NewStyle().Foreground(ColDim)
	sRec          = lipgloss.NewStyle().Foreground(ColRed).Bold(true)
	sSectionTitle = lipgloss.NewStyle().Foreground(ColPaper).Bold(true)
	sSectionKey   = lipgloss.NewStyle().Foreground(ColAmber).Bold(true)

	sFooterKey  = lipgloss.NewStyle().Foreground(ColAmber).Bold(true)
	sFooterText = lipgloss.NewStyle().Foreground(ColDim)

	sMastTitle = lipgloss.NewStyle().Foreground(ColAmber).Bold(true)
	sMastClock = lipgloss.NewStyle().Foreground(ColPaper).Bold(true)

	sCursor     = lipgloss.NewStyle().Foreground(ColAmber).Bold(true)
	sCursorBody = lipgloss.NewStyle().Foreground(ColPaper).Bold(true)
	sURL        = lipgloss.NewStyle().Foreground(ColCyan).Underline(true)
	sSnippet    = lipgloss.NewStyle().Foreground(ColText)
	sPromptMark = lipgloss.NewStyle().Foreground(ColAmber).Bold(true)
)

// latencyPill classifies a request duration.
func latencyPill(ms int64) string {
	switch {
	case ms < 0:
		return sDim.Render("[  —   ]")
	case ms < 250:
		return sGood.Render("[ FAST ]")
	case ms < 750:
		return sAmber.Render("[  OK  ]")
	case ms < 2000:
		return sHot.Render("[ SLOW ]")
	default:
		return sCrit.Render("[ LAG  ]")
	}
}

// statePill reports overall engine outcome.
func statePill(ok bool, empty bool) string {
	switch {
	case !ok:
		return sCrit.Render("[ ERR  ]")
	case empty:
		return sAmber.Render("[ VOID ]")
	default:
		return sGood.Render("[ NOM  ]")
	}
}
