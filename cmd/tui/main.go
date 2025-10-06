package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/quniob/shellm/agent"
	"github.com/quniob/shellm/config"
	"github.com/quniob/shellm/tools"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func waitForAgentMsg(msgCh chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-msgCh
		if !ok {
			return nil
		}
		return msg
	}
}

func runAgent(timeout int, ag *agent.Agent, userInput string, msgCh chan tea.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
	go func() {
		defer close(msgCh)
		defer cancel()
		ag.Start(ctx, userInput, msgCh)
	}()
}

type ChatMessage struct {
	sender  string
	content string
	style   lipgloss.Style
}

type model struct {
	logView      viewport.Model
	chatView     viewport.Model
	textarea     textarea.Model
	spinner      spinner.Model
	thinking     bool
	logMessages  []ChatMessage
	chatMessages []ChatMessage
	Agent        *agent.Agent
	Config       *config.Config
	tokenUsage   int
	messagesChan chan tea.Msg
	userStyle    lipgloss.Style
	agentStyle   lipgloss.Style
	toolStyle    lipgloss.Style
	thoughtStyle lipgloss.Style
	errorStyle   lipgloss.Style
}

func (m *model) renderLogMessages() {
	var result strings.Builder
	result.WriteString("\n")
	for _, msg := range m.logMessages {
		result.WriteString(msg.style.Render(msg.sender) + msg.content + "\n")
	}
	m.logView.SetContent(lipgloss.NewStyle().Width(m.logView.Width).Padding(1).Render(result.String()))
	m.logView.GotoBottom()
}

func (m *model) renderChatMessages() {
	var result strings.Builder
	result.WriteString("\n")
	for _, msg := range m.chatMessages {
		result.WriteString(msg.style.Render(msg.sender) + msg.content + "\n")
		if msg.sender == "󰚩 :" {
			separator := lipgloss.NewStyle().
				Width(m.chatView.Width - 2).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(lipgloss.Color("240")).
				PaddingLeft(1).PaddingRight(1).
				String()
			result.WriteString(separator + "\n")
		}
	}
	m.chatView.SetContent(lipgloss.NewStyle().Width(m.chatView.Width).Padding(1).Render(result.String()))
	m.chatView.GotoBottom()
}

func initialModel() model {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return model{}
	}
	hosts, err := config.LoadHosts(cfg.InventoryPath, cfg.SecretsPath)
	if err != nil {
		fmt.Println("Error loading hosts:", err)
		return model{}
	}
	reg := tools.NewRegistry(tools.Report{}, tools.Ping{}, tools.GetHosts{HostsData: &hosts}, tools.ExecuteCommand{HostsData: &hosts})
	ag := agent.NewAgent(reg, cfg)

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.Prompt = "> "
	ta.CharLimit = 0
	ta.SetHeight(1)
	ta.KeyMap.InsertNewline.SetEnabled(false)
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		PaddingLeft(1).
		PaddingRight(1)
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		PaddingLeft(1).
		PaddingRight(1).
		Foreground(lipgloss.Color("8"))
	ta.ShowLineNumbers = false

	lv := viewport.New(40, 20)
	lv.SetContent("Logs")
	cv := viewport.New(40, 20)
	cv.SetContent("Welcome to shellm! Ask me anything.")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		textarea:     ta,
		logView:      lv,
		chatView:     cv,
		spinner:      s,
		logMessages:  []ChatMessage{},
		chatMessages: []ChatMessage{},
		Agent:        ag,
		Config:       cfg,
		tokenUsage:   0,
		userStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
		agentStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
		toolStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		thoughtStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true),
		errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd  tea.Cmd
		lvCmd  tea.Cmd
		cvCmd  tea.Cmd
		spCmd  tea.Cmd
		vpCmds tea.Cmd
	)

	if msg == nil {
		return m, nil
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.logView, lvCmd = m.logView.Update(msg)
	m.chatView, cvCmd = m.chatView.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	vpCmds = tea.Batch(lvCmd, cvCmd)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		separatorWidth := 1
		m.logView.Width = msg.Width / 3
		m.chatView.Width = msg.Width - m.logView.Width - separatorWidth
		m.textarea.SetWidth(msg.Width)
		m.logView.Height = msg.Height - m.textarea.Height() - lipgloss.Height("\n")
		m.chatView.Height = msg.Height - m.textarea.Height() - lipgloss.Height("\n")
		m.renderLogMessages()
		m.renderChatMessages()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.thinking {
				return m, nil
			}
			userInput := m.textarea.Value() + "\n"
			if userInput == "" {
				return m, nil
			}
			userMessage := ChatMessage{sender: " : ", content: userInput, style: m.userStyle}
			m.chatMessages = append(m.chatMessages, userMessage)
			m.renderLogMessages()
			m.renderChatMessages()
			m.textarea.Reset()

			m.thinking = true
			m.textarea.Blur()

			m.messagesChan = make(chan tea.Msg)
			runAgent(m.Config.LLMTimeOut, m.Agent, userInput, m.messagesChan)

			return m, waitForAgentMsg(m.messagesChan)
		}

	case agent.TokenUsageMsg:
		m.tokenUsage = msg.Tokens
		return m, waitForAgentMsg(m.messagesChan)
	case agent.ThoughtMsg:
		m.logMessages = append(m.logMessages, ChatMessage{sender: "Thought: ", content: msg.Content, style: m.thoughtStyle})
		m.renderLogMessages()
		return m, waitForAgentMsg(m.messagesChan)
	case agent.ToolCallMsg:
		m.logMessages = append(m.logMessages, ChatMessage{sender: "Tool Call: ", content: msg.Content, style: m.toolStyle})
		m.renderLogMessages()
		return m, waitForAgentMsg(m.messagesChan)
	case agent.ToolResultMsg:
		m.logMessages = append(m.logMessages, ChatMessage{sender: "Tool Result: ", content: msg.Content, style: m.toolStyle})
		m.renderLogMessages()
		return m, waitForAgentMsg(m.messagesChan)
	case agent.FinalResultMsg:
		r, _ := glamour.NewTermRenderer(
			glamour.WithStylePath("dark"),
			glamour.WithWordWrap(m.chatView.Width),
		)
		out, err := r.Render(msg.Content)
		if err != nil {
			out = msg.Content
		}
		out = strings.TrimPrefix(out, "\n")
		out = strings.TrimSuffix(out, "\n")
		out = strings.TrimSuffix(out, "\n")

		agentMessage := ChatMessage{sender: "󰚩 :", content: out, style: m.agentStyle}
		m.chatMessages = append(m.chatMessages, agentMessage)
		m.thinking = false
		m.textarea.Placeholder = "Send a message..."
		m.textarea.Focus()
		m.renderLogMessages()
		m.renderChatMessages()
		return m, nil
	case agent.ErrMsg:
		m.logMessages = append(m.logMessages, ChatMessage{sender: "Error: ", content: msg.Err.Error(), style: m.errorStyle})
		m.thinking = false
		m.textarea.Placeholder = "Send a message..."
		m.textarea.Focus()
		m.renderLogMessages()

		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmds, spCmd)
}

func (m model) View() string {
	var thinkingIndicator string
	if m.thinking {
		thinkingIndicator = m.spinner.View() + " Agent is thinking..."
	}

	tokenUsageIndicator := fmt.Sprintf("Tokens usage: %d", m.tokenUsage)

	separator := lipgloss.NewStyle().
		Height(m.logView.Height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("240")).
		String()

	mainView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.logView.View(),
		separator,
		m.chatView.View(),
	)

	return fmt.Sprintf(
		"%s\n%s\n%s\n%s",
		mainView,
		m.textarea.View(),
		tokenUsageIndicator,
		thinkingIndicator,
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}
