package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Fixed vertical chrome around the scrollable message area.
const (
	headerHeight   = 1
	footerHeight   = 1
	composerInner  = 3 // textarea rows
	composerborder = 2 // rounded border, top + bottom
)

var (
	headerBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("231")).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	userBubbleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("231")).
			Padding(0, 1)

	nomiBubbleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("238")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	errorBubbleStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("88")).
				Foreground(lipgloss.Color("231")).
				Padding(0, 1)

	captionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	typingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	placeholderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Italic(true)

	composerBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63"))

	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("213")).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("213"))

	modalSelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("63"))

	modalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	modalDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

// message is a single rendered line in a conversation transcript.
type message struct {
	fromUser bool
	isError  bool
	text     string
}

// conversation holds the in-memory transcript for one Nomi.
type conversation struct {
	nomi     Nomi
	messages []message
	pending  bool // a reply is in flight
}

// replyMsg is delivered when an async SendMessage call completes. It carries
// the Nomi ID so a reply is routed to the right conversation even if the user
// has switched Nomis in the meantime.
type replyMsg struct {
	nomiID string
	reply  string
	err    error
}

// chatTUI is the unified BubbleTea model: a chat view plus a Nomi switcher.
type chatTUI struct {
	client *NomiClient
	nomis  []Nomi
	convos map[string]*conversation // keyed by Nomi UUID
	active string                   // UUID of the Nomi currently in view

	viewport viewport.Model
	composer textarea.Model
	spin     spinner.Model

	switcherOpen bool
	filter       textinput.Model
	switcherCur  int

	width, height int
	ready         bool
}

func newChatTUI(nomis []Nomi) chatTUI {
	ta := textarea.New()
	ta.Placeholder = "Type a message…"
	ta.Prompt = "┃ "
	ta.ShowLineNumbers = false
	ta.CharLimit = 4000
	ta.SetHeight(composerInner)
	ta.Focus()

	fi := textinput.New()
	fi.Placeholder = "filter…"
	fi.Prompt = "🔍 "
	fi.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))

	return chatTUI{
		client:   client,
		nomis:    nomis,
		convos:   make(map[string]*conversation),
		viewport: viewport.New(0, 0),
		composer: ta,
		filter:   fi,
		spin:     sp,
	}
}

func (m chatTUI) Init() tea.Cmd {
	return textarea.Blink
}

func (m chatTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.applyLayout()
		m.ready = true
		m.refreshViewport()
		m.viewport.GotoBottom()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		if m.anyPending() {
			m.refreshViewport()
			return m, cmd
		}
		return m, nil

	case replyMsg:
		m.handleReply(msg)
		return m, nil

	case tea.KeyMsg:
		if m.switcherOpen {
			return m.updateSwitcher(msg)
		}
		return m.updateChat(msg)
	}
	return m, nil
}

func (m chatTUI) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "tab":
		m.openSwitcher()
		return m, textinput.Blink
	case "enter":
		return m, m.send()
	case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	default:
		var cmd tea.Cmd
		m.composer, cmd = m.composer.Update(msg)
		return m, cmd
	}
}

func (m chatTUI) updateSwitcher(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "tab":
		m.closeSwitcher()
		return m, nil
	case "up", "ctrl+p":
		if m.switcherCur > 0 {
			m.switcherCur--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.switcherCur < len(m.filteredNomis())-1 {
			m.switcherCur++
		}
		return m, nil
	case "enter":
		if filtered := m.filteredNomis(); len(filtered) > 0 {
			m.selectNomi(filtered[m.switcherCur].UUID)
		}
		m.closeSwitcher()
		return m, nil
	default:
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		if n := len(m.filteredNomis()); m.switcherCur >= n {
			m.switcherCur = max(0, n-1)
		}
		return m, cmd
	}
}

// send dispatches the composer's text to the active Nomi asynchronously.
func (m *chatTUI) send() tea.Cmd {
	text := strings.TrimSpace(m.composer.Value())
	if text == "" || m.active == "" {
		return nil
	}
	wasPending := m.anyPending()
	conv := m.convos[m.active]
	conv.messages = append(conv.messages, message{fromUser: true, text: text})
	conv.pending = true
	m.composer.Reset()
	m.refreshViewport()
	m.viewport.GotoBottom()

	cmd := sendCmd(m.client, m.active, text)
	if wasPending {
		return cmd // spinner is already ticking
	}
	return tea.Batch(cmd, m.spin.Tick)
}

// sendCmd performs the blocking API call off the UI thread.
func sendCmd(c *NomiClient, nomiID, text string) tea.Cmd {
	return func() tea.Msg {
		resp, err := c.SendMessage(nomiID, text)
		if err != nil {
			return replyMsg{nomiID: nomiID, err: err}
		}
		return replyMsg{nomiID: nomiID, reply: resp.ReplyMessage.Text}
	}
}

func (m *chatTUI) handleReply(msg replyMsg) {
	conv := m.convos[msg.nomiID]
	if conv == nil {
		return
	}
	conv.pending = false
	if msg.err != nil {
		conv.messages = append(conv.messages, message{isError: true, text: "Error: " + msg.err.Error()})
	} else {
		conv.messages = append(conv.messages, message{text: msg.reply})
	}
	if msg.nomiID == m.active {
		m.refreshViewport()
		m.viewport.GotoBottom()
	}
}

func (m *chatTUI) openSwitcher() {
	m.switcherOpen = true
	m.composer.Blur()
	m.filter.SetValue("")
	m.filter.Focus()
	m.switcherCur = 0
	for i, n := range m.nomis {
		if n.UUID == m.active {
			m.switcherCur = i
		}
	}
}

func (m *chatTUI) closeSwitcher() {
	m.switcherOpen = false
	m.filter.Blur()
	m.composer.Focus()
}

// selectNomi makes uuid the active conversation, creating it if needed.
func (m *chatTUI) selectNomi(uuid string) {
	if _, ok := m.convos[uuid]; !ok {
		for _, n := range m.nomis {
			if n.UUID == uuid {
				m.convos[uuid] = &conversation{nomi: n}
			}
		}
	}
	if _, ok := m.convos[uuid]; !ok {
		return
	}
	m.active = uuid
	m.refreshViewport()
	m.viewport.GotoBottom()
}

func (m chatTUI) filteredNomis() []Nomi {
	q := strings.ToLower(strings.TrimSpace(m.filter.Value()))
	if q == "" {
		return m.nomis
	}
	var out []Nomi
	for _, n := range m.nomis {
		if strings.Contains(strings.ToLower(n.Name), q) {
			out = append(out, n)
		}
	}
	return out
}

func (m chatTUI) anyPending() bool {
	for _, c := range m.convos {
		if c.pending {
			return true
		}
	}
	return false
}

func (m *chatTUI) applyLayout() {
	vpHeight := m.height - headerHeight - footerHeight - composerInner - composerborder
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.viewport.Width = m.width
	m.viewport.Height = vpHeight
	m.composer.SetWidth(m.width - 2)
	m.composer.SetHeight(composerInner)
}

func (m *chatTUI) refreshViewport() {
	m.viewport.SetContent(m.renderConversation())
}

// renderConversation builds the scrollable transcript for the active Nomi.
func (m chatTUI) renderConversation() string {
	conv := m.convos[m.active]
	if conv == nil {
		return placeholderStyle.Render("\n  Press Tab to pick a Nomi.")
	}
	w := m.viewport.Width
	if w <= 0 {
		w = m.width
	}
	if len(conv.messages) == 0 && !conv.pending {
		return placeholderStyle.Render("\n  No messages yet — say hello to " + conv.nomi.Name + " 👋")
	}

	maxBubble := w * 3 / 4
	if maxBubble < 12 {
		maxBubble = 12
	}

	var b strings.Builder
	for i, msg := range conv.messages {
		if i > 0 {
			b.WriteString("\n\n")
		}
		style, pos, caption := nomiBubbleStyle, lipgloss.Left, conv.nomi.Name
		switch {
		case msg.isError:
			style, pos, caption = errorBubbleStyle, lipgloss.Left, "error"
		case msg.fromUser:
			style, pos, caption = userBubbleStyle, lipgloss.Right, "You"
		}
		bub := renderBubble(style, msg.text, maxBubble)
		b.WriteString(lipgloss.PlaceHorizontal(w, pos, captionStyle.Render(caption)))
		b.WriteString("\n")
		b.WriteString(lipgloss.PlaceHorizontal(w, pos, bub))
	}
	if conv.pending {
		if len(conv.messages) > 0 {
			b.WriteString("\n\n")
		}
		line := typingStyle.Render(m.spin.View() + " " + conv.nomi.Name + " is typing…")
		b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Left, line))
	}
	return b.String()
}

// renderBubble renders text in style, wrapping only once it exceeds max width.
func renderBubble(style lipgloss.Style, text string, max int) string {
	if lipgloss.Width(text) > max {
		style = style.Width(max)
	}
	return style.Render(text)
}

func (m chatTUI) View() string {
	if !m.ready || m.width == 0 {
		return "Initializing…"
	}
	if m.switcherOpen {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, m.switcherView())
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerView(),
		m.viewport.View(),
		m.composerView(),
		m.footerView(),
	)
}

func (m chatTUI) headerView() string {
	label := "No Nomi selected"
	if conv := m.convos[m.active]; conv != nil {
		label = conv.nomi.Name
		if conv.nomi.RelationshipType != "" {
			label += " · " + conv.nomi.RelationshipType
		}
	}
	return headerBarStyle.Width(m.width).Render(" 💬 " + label)
}

func (m chatTUI) composerView() string {
	return composerBoxStyle.Width(m.width - 2).Render(m.composer.View())
}

func (m chatTUI) footerView() string {
	return footerStyle.Width(m.width).
		Render(" Tab switch Nomi · Enter send · PgUp/PgDn scroll · Ctrl+C quit")
}

// switcherView renders the centered Nomi-picker modal.
func (m chatTUI) switcherView() string {
	innerW := 46
	if m.width-6 < innerW {
		innerW = m.width - 6
	}
	if innerW < 16 {
		innerW = 16
	}

	filtered := m.filteredNomis()
	var b strings.Builder
	b.WriteString(modalTitleStyle.Render(fmt.Sprintf("Switch Nomi (%d)", len(filtered))))
	b.WriteString("\n")
	b.WriteString(m.filter.View())
	b.WriteString("\n\n")

	if len(filtered) == 0 {
		b.WriteString(modalDimStyle.Render("no matches"))
	}

	// Window the list so it never overflows the screen vertically.
	maxVisible := m.height - 10
	if maxVisible < 3 {
		maxVisible = 3
	}
	start := 0
	if m.switcherCur >= maxVisible {
		start = m.switcherCur - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := start; i < end; i++ {
		n := filtered[i]
		row := n.Name
		if n.RelationshipType != "" {
			row += "  (" + n.RelationshipType + ")"
		}
		if i == m.switcherCur {
			b.WriteString(modalSelStyle.Width(innerW).Render("▸ " + row))
		} else {
			b.WriteString(modalItemStyle.Width(innerW).Render("  " + row))
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	b.WriteString(modalDimStyle.Render("↑/↓ move · Enter select · Esc cancel"))
	return modalBoxStyle.Render(b.String())
}

// runTUI launches the interactive chat. If focusName is non-empty the chat
// opens directly on that Nomi; otherwise it opens with the switcher in front.
func runTUI(nomis []Nomi, focusName string) error {
	if len(nomis) == 0 {
		return fmt.Errorf("no Nomis found")
	}
	m := newChatTUI(nomis)

	if focusName != "" {
		var match string
		for _, n := range nomis {
			if strings.EqualFold(n.Name, focusName) {
				match = n.UUID
			}
		}
		if match == "" {
			return fmt.Errorf("no Nomi found with the name: %s", focusName)
		}
		m.selectNomi(match)
	} else {
		m.selectNomi(nomis[0].UUID)
		m.openSwitcher()
	}

	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
