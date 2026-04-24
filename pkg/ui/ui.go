package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"atlas.websearch/pkg/search"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
)

// --- messages ---------------------------------------------------------------

type tickMsg time.Time

type searchDoneMsg struct {
	query    string
	engine   string
	outcomes []search.Outcome
	results  []search.Result
	elapsed  time.Duration
}

// --- focus ------------------------------------------------------------------

type focusMode int

const (
	focusList focusMode = iota
	focusInput
)

// --- model ------------------------------------------------------------------

type model struct {
	version string
	engines []search.Engine
	engine  int  // index into engines, or len(engines) for "ALL"
	limit   int
	input   textinput.Model

	focus    focusMode
	loading  bool
	err      error
	query    string
	results  []search.Result
	cursor   int
	top      int           // scroll offset in result list
	outcomes []search.Outcome
	elapsed  time.Duration

	width, height int
	frame         int
	blink         bool
	started       time.Time

	pendingCancel context.CancelFunc
}

// Config bundles launch parameters for the UI.
type Config struct {
	Version      string
	InitialQuery string
	EngineCode   string // "all" or an engine code
	Limit        int
}

func newModel(cfg Config) model {
	engines := search.Registry()

	ti := textinput.New()
	ti.Placeholder = "type a query and press ↵"
	ti.Prompt = ""
	ti.CharLimit = 256
	ti.TextStyle = sCursorBody
	ti.PlaceholderStyle = sDim
	ti.SetValue(cfg.InitialQuery)

	m := model{
		version: cfg.Version,
		engines: engines,
		limit:   cfg.Limit,
		input:   ti,
		started: time.Now(),
	}

	m.engine = engineIndex(engines, cfg.EngineCode)

	if strings.TrimSpace(cfg.InitialQuery) == "" {
		m.focus = focusInput
		m.input.Focus()
	} else {
		m.focus = focusList
	}
	return m
}

func engineIndex(engines []search.Engine, code string) int {
	if code == "all" {
		return len(engines)
	}
	for i, e := range engines {
		if e.Code() == code {
			return i
		}
	}
	return 0
}

func (m model) currentEngineLabel() string {
	if m.engine == len(m.engines) {
		return "ALL ENGINES"
	}
	return m.engines[m.engine].Name()
}

// --- tea.Model --------------------------------------------------------------

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, heartbeat(), textinput.Blink)
	if strings.TrimSpace(m.input.Value()) != "" {
		cmds = append(cmds, m.runSearch(m.input.Value()))
	}
	return tea.Batch(cmds...)
}

func heartbeat() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) cancelInFlight() {
	if m.pendingCancel != nil {
		m.pendingCancel()
		m.pendingCancel = nil
	}
}

func (m *model) runSearch(query string) tea.Cmd {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	m.cancelInFlight()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	m.pendingCancel = cancel
	m.loading = true
	m.err = nil
	m.results = nil
	m.outcomes = nil
	m.cursor = 0
	m.top = 0
	m.query = query

	limit := m.limit
	if limit <= 0 {
		limit = 10
	}

	engineIdx := m.engine
	engines := m.engines

	return func() tea.Msg {
		defer cancel()
		opts := search.Options{Query: query, Limit: limit}
		start := time.Now()

		if engineIdx == len(engines) {
			outs := search.RunAll(ctx, engines, opts)
			return searchDoneMsg{
				query:    query,
				engine:   "ALL ENGINES",
				outcomes: outs,
				results:  search.Merge(outs),
				elapsed:  time.Since(start),
			}
		}

		e := engines[engineIdx]
		resp, err := e.Search(ctx, opts)
		out := search.Outcome{Engine: e, Response: resp, Err: err, Latency: time.Since(start)}
		var results []search.Result
		if resp != nil {
			results = resp.Results
			for i := range results {
				results[i].Source = e.Name()
			}
		}
		return searchDoneMsg{
			query:    query,
			engine:   e.Name(),
			outcomes: []search.Outcome{out},
			results:  results,
			elapsed:  time.Since(start),
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, tea.ClearScreen

	case tickMsg:
		m.frame++
		m.blink = m.frame%2 == 0
		return m, heartbeat()

	case searchDoneMsg:
		m.loading = false
		m.query = msg.query
		m.results = msg.results
		m.outcomes = msg.outcomes
		m.elapsed = msg.elapsed
		// Record engine-level errors: if all engines errored, surface it.
		if len(msg.results) == 0 {
			allErr := true
			for _, o := range msg.outcomes {
				if o.Err == nil {
					allErr = false
					break
				}
			}
			if allErr && len(msg.outcomes) > 0 {
				m.err = msg.outcomes[0].Err
			}
		}
		m.focus = focusList
		m.input.Blur()
		return m, nil

	case tea.KeyMsg:
		if m.focus == focusInput {
			return m.updateInput(msg)
		}
		return m.updateList(msg)
	}

	return m, nil
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if strings.TrimSpace(m.input.Value()) == "" && m.query == "" {
			return m, tea.Quit
		}
		m.focus = focusList
		m.input.Blur()
		return m, nil
	case "enter":
		q := strings.TrimSpace(m.input.Value())
		if q == "" {
			return m, nil
		}
		cmd := m.runSearch(q)
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.cancelInFlight()
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
	case "g", "home":
		m.cursor = 0
		m.top = 0
	case "G", "end":
		if len(m.results) > 0 {
			m.cursor = len(m.results) - 1
		}
	case "enter", "o":
		if m.cursor >= 0 && m.cursor < len(m.results) {
			_ = browser.OpenURL(m.results[m.cursor].URL)
		}
	case "/":
		m.focus = focusInput
		m.input.Focus()
		m.input.SetValue(m.query)
		m.input.CursorEnd()
		return m, textinput.Blink
	case "e":
		// Cycle engine forward and re-run if we have a query.
		m.engine = (m.engine + 1) % (len(m.engines) + 1)
		if strings.TrimSpace(m.query) != "" {
			return m, m.runSearch(m.query)
		}
	case "E":
		m.engine = (m.engine - 1 + len(m.engines) + 1) % (len(m.engines) + 1)
		if strings.TrimSpace(m.query) != "" {
			return m, m.runSearch(m.query)
		}
	case "r":
		if strings.TrimSpace(m.query) != "" {
			return m, m.runSearch(m.query)
		}
	}
	return m, nil
}

// --- View -------------------------------------------------------------------

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.width < 64 {
		return sCrit.Render(" terminal too narrow — resize to ≥ 64 columns ")
	}

	var blocks []string
	blocks = append(blocks, m.renderMasthead())
	blocks = append(blocks, m.renderConsole(m.width))
	blocks = append(blocks, m.renderResults(m.width))
	blocks = append(blocks, m.renderEngines(m.width))

	body := strings.Join(blocks, "\n")
	full := body + "\n" + m.renderFooter()

	lines := strings.Split(full, "\n")
	available := m.height
	if len(lines) < available {
		blank := strings.Repeat(" ", m.width)
		for len(lines) < available {
			lines = append(lines, blank)
		}
	} else if len(lines) > available {
		lines = lines[:available]
	}
	return strings.Join(lines, "\n")
}

// --- Masthead ---------------------------------------------------------------

func (m model) renderMasthead() string {
	w := m.width
	rule := sBorder.Render(strings.Repeat("━", w))

	title := sMastTitle.Render("A T L A S") +
		sDim.Render("  ·  ") +
		sMastTitle.Render("W E B S E A R C H")

	clock := sMastClock.Render(time.Now().Format("15:04:05"))
	var rec string
	switch {
	case m.loading:
		rec = sHot.Render("◼ BUSY")
	case m.blink:
		rec = sRec.Render("● ONLN")
	default:
		rec = sDim.Render("● ONLN")
	}
	ver := sDim.Render("v" + m.version)
	right := horiz(clock, rec, ver)

	titleW := lipgloss.Width(title)
	rightW := lipgloss.Width(right)
	pad := w - 2 - titleW - rightW
	if pad < 1 {
		pad = 1
	}
	line1 := "  " + title + strings.Repeat(" ", pad) + right

	qTxt := nonempty(m.query, "—")
	engineTxt := m.currentEngineLabel()
	count := fmt.Sprintf("%d", len(m.results))
	lat := "—"
	if m.elapsed > 0 {
		lat = m.elapsed.Truncate(time.Millisecond).String()
	}

	meta := horiz(
		sDim.Render("QUERY ")+sValue.Render(truncateVisible(qTxt, w/3)),
		sDim.Render("ENGINE ")+sValue.Render(engineTxt),
		sDim.Render("HITS ")+sValue.Render(count),
		sDim.Render("LATENCY ")+sValue.Render(lat),
	)
	line2 := "  " + meta
	if lipgloss.Width(line2) > w {
		meta = horiz(
			sDim.Render("QUERY ")+sValue.Render(truncateVisible(qTxt, w/4)),
			sDim.Render("ENGINE ")+sValue.Render(engineTxt),
			sDim.Render("HITS ")+sValue.Render(count),
		)
		line2 = "  " + meta
	}

	return strings.Join([]string{rule, line1, line2, rule}, "\n")
}

// --- Console (search input) -------------------------------------------------

func (m model) renderConsole(w int) string {
	inner := w - 4
	mark := sPromptMark.Render("❯ ")
	markW := lipgloss.Width(mark)

	var body string
	if m.focus == focusInput {
		m.input.Width = inner - markW - 2
		body = mark + m.input.View()
	} else {
		shown := nonempty(m.query, sDim.Render("press / to enter a query"))
		if m.query != "" {
			shown = sCursorBody.Render(shown)
		}
		body = mark + truncateVisible(shown, inner-markW)
	}

	hint := ""
	if m.loading {
		hint = sHot.Render(spinner(m.frame) + " searching…")
	} else if m.err != nil {
		hint = sCrit.Render("✕ " + truncateVisible(m.err.Error(), inner/2))
	} else if len(m.results) == 0 && m.query != "" {
		hint = sAmber.Render("◌ no results")
	}

	if hint != "" {
		pad := inner - lipgloss.Width(body) - lipgloss.Width(hint)
		if pad < 1 {
			pad = 1
		}
		body = body + strings.Repeat(" ", pad) + hint
	}

	return section("01", "CONSOLE", body, w)
}

func spinner(frame int) string {
	frames := []string{"◐", "◓", "◑", "◒"}
	return frames[frame%len(frames)]
}

// --- Results ----------------------------------------------------------------

func (m *model) ensureCursorVisible(visibleRows int) {
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+visibleRows {
		m.top = m.cursor - visibleRows + 1
	}
	if m.top < 0 {
		m.top = 0
	}
}

func (m model) renderResults(w int) string {
	if len(m.results) == 0 {
		var body string
		switch {
		case m.loading:
			body = sDim.Render("awaiting response…")
		case m.query == "":
			body = sDim.Render("no query yet — press / to start")
		default:
			body = sDim.Render("no results for \"" + m.query + "\"")
		}
		return section("02", "RESULTS", body, w)
	}

	inner := w - 4
	// 3 rows per result + 1 gap = 4; masthead(4) + console(3) + engines(~6) + footer(1) ≈ 14 overhead.
	perItem := 4
	reserved := 22
	avail := m.height - reserved
	if avail < perItem {
		avail = perItem
	}
	visibleItems := avail / perItem
	if visibleItems < 1 {
		visibleItems = 1
	}
	if visibleItems > len(m.results) {
		visibleItems = len(m.results)
	}

	// Scroll adjustment (via pointer receiver semantics on a copy).
	mm := &m
	mm.ensureCursorVisible(visibleItems)

	rankW := 3
	sourceW := 0
	for _, r := range m.results {
		if lipgloss.Width(r.Source) > sourceW {
			sourceW = lipgloss.Width(r.Source)
		}
	}
	if sourceW > 12 {
		sourceW = 12
	}

	var rows []string
	end := mm.top + visibleItems
	if end > len(m.results) {
		end = len(m.results)
	}
	for i := mm.top; i < end; i++ {
		r := m.results[i]
		selected := i == mm.cursor

		rank := sDim.Render(fmt.Sprintf("%02d", i+1))
		marker := "  "
		titleStyle := sCursorBody
		if selected {
			marker = sCursor.Render("❯ ")
			titleStyle = sAmber
		}

		titleMax := inner - rankW - 1 - lipgloss.Width(marker) - sourceW - 2
		if titleMax < 10 {
			titleMax = 10
		}
		title := titleStyle.Render(truncateVisible(r.Title, titleMax))

		srcLabel := ""
		if sourceW > 0 && r.Source != "" {
			srcLabel = sDim.Render(padRight(truncateVisible(r.Source, sourceW), sourceW))
		} else if sourceW > 0 {
			srcLabel = strings.Repeat(" ", sourceW)
		}

		line1 := rank + " " + marker + title
		line1W := lipgloss.Width(line1) + lipgloss.Width(srcLabel)
		padGap := inner - line1W
		if padGap < 1 {
			padGap = 1
		}
		line1 = line1 + strings.Repeat(" ", padGap) + srcLabel

		urlMax := inner - 6
		url := sURL.Render(truncateVisible(r.URL, urlMax))
		line2 := "     " + url

		snipMax := inner - 6
		snip := sSnippet.Render(truncateVisible(firstLine(r.Snippet), snipMax))
		line3 := "     " + snip

		rows = append(rows, line1, line2, line3)
		if i < end-1 {
			rows = append(rows, "")
		}
	}

	// Footer line inside the box: "showing X–Y of N".
	rows = append(rows, "")
	footer := fmt.Sprintf("showing %d–%d of %d", mm.top+1, end, len(m.results))
	rows = append(rows, sDim.Render(footer))

	return section("02", "RESULTS", strings.Join(rows, "\n"), w)
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// --- Engines (per-engine telemetry) ----------------------------------------

func (m model) renderEngines(w int) string {
	if len(m.outcomes) == 0 {
		all := m.engines
		var rows []string
		for i, e := range all {
			sel := i == m.engine
			rows = append(rows, engineRow(e, search.Outcome{Engine: e}, sel, false, w-4))
		}
		allSel := m.engine == len(all)
		rows = append(rows, engineHeader("ALL ENGINES", allSel, w-4))
		return section("03", "ENGINES", strings.Join(rows, "\n"), w)
	}

	var rows []string
	for i, e := range m.engines {
		var o search.Outcome
		o.Engine = e
		for _, oc := range m.outcomes {
			if oc.Engine != nil && oc.Engine.Code() == e.Code() {
				o = oc
				break
			}
		}
		rows = append(rows, engineRow(e, o, i == m.engine, true, w-4))
	}
	allSel := m.engine == len(m.engines)
	rows = append(rows, engineHeader("ALL ENGINES", allSel, w-4))
	return section("03", "ENGINES", strings.Join(rows, "\n"), w)
}

func engineHeader(label string, selected bool, inner int) string {
	marker := "  "
	lbl := sDim.Render(label)
	if selected {
		marker = sCursor.Render("❯ ")
		lbl = sAmber.Render(label)
	}
	line := marker + lbl
	return padLeft(line, inner)
}

func engineRow(e search.Engine, o search.Outcome, selected bool, hasData bool, inner int) string {
	marker := "  "
	name := sText.Render(padLeft(e.Name(), 14))
	code := sDim.Render(padLeft("["+e.Code()+"]", 10))
	if selected {
		marker = sCursor.Render("❯ ")
		name = sAmber.Render(padLeft(e.Name(), 14))
	}

	var state, lat, hits string
	if !hasData {
		state = statePill(true, true)
		lat = sDim.Render(padRight("—", 8))
		hits = sDim.Render(padRight("— hits", 9))
	} else if o.Err != nil {
		state = statePill(false, true)
		lat = sDim.Render(padRight(o.Latency.Truncate(time.Millisecond).String(), 8))
		hits = sCrit.Render(padRight("ERR", 9))
	} else {
		n := 0
		if o.Response != nil {
			n = len(o.Response.Results)
		}
		state = statePill(true, n == 0)
		lat = sValue.Render(padRight(o.Latency.Truncate(time.Millisecond).String(), 8))
		hits = sValue.Render(padRight(fmt.Sprintf("%d hits", n), 9))
	}

	latPill := latencyPill(int64(o.Latency / time.Millisecond))
	if !hasData {
		latPill = latencyPill(-1)
	}

	line := marker + name + "  " + code + "  " + state + "  " + latPill + "  " + lat + "  " + hits
	return padLeft(line, inner)
}

// --- Footer -----------------------------------------------------------------

func (m model) renderFooter() string {
	var keys []string
	if m.focus == focusInput {
		keys = []string{
			sFooterKey.Render("[↵]") + sFooterText.Render("·RUN"),
			sFooterKey.Render("[ESC]") + sFooterText.Render("·CANCEL"),
			sFooterKey.Render("[CTRL-C]") + sFooterText.Render("·QUIT"),
		}
	} else {
		keys = []string{
			sFooterKey.Render("[↑↓ J/K]") + sFooterText.Render("·NAV"),
			sFooterKey.Render("[↵]") + sFooterText.Render("·OPEN"),
			sFooterKey.Render("[/]") + sFooterText.Render("·SEARCH"),
			sFooterKey.Render("[E]") + sFooterText.Render("·ENGINE"),
			sFooterKey.Render("[R]") + sFooterText.Render("·RERUN"),
			sFooterKey.Render("[Q]") + sFooterText.Render("·QUIT"),
		}
	}
	left := " " + strings.Join(keys, "   ")
	right := sDim.Render(fmt.Sprintf(" uptime · %s ", time.Since(m.started).Truncate(time.Second)))
	pad := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

// --- Entry point ------------------------------------------------------------

// Start launches the interactive UI.
func Start(cfg Config) error {
	p := tea.NewProgram(newModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
