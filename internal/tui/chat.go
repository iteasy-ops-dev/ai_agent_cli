package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/iteasy-ops-dev/syseng-agent/internal/agent"
	"github.com/iteasy-ops-dev/syseng-agent/pkg/types"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86")).
				Bold(true).
				Margin(1, 0)

	agentMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("39")).
				Margin(1, 0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	viewportStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1)
)

type ChatModel struct {
	viewport      viewport.Model
	textarea      textarea.Model
	agent         *agent.Agent
	session       *types.ConversationSession
	conversation  []ConversationEntry
	processing    bool
	err           error
	width         int
	height        int
}

type ConversationEntry struct {
	Type    string // "user" or "agent" or "error"
	Message string
	Error   string
}

type AgentResponseMsg struct {
	Message string
	Error   string
	Err     error
}

func NewChatModel(ag *agent.Agent, mcpServerID, providerID string, interactive bool) ChatModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message here... (Enter to send, Esc to quit)"
	ta.Focus()
	ta.SetHeight(3)
	ta.CharLimit = 2000

	vp := viewport.New(80, 20)
	vp.SetContent("🤖 Welcome to the AI Agent Chat!\nType your message below and press Ctrl+Enter to send.\n")

	// Create conversation session
	session := &types.ConversationSession{
		ID:          uuid.New().String(),
		MCPServerID: mcpServerID,
		ProviderID:  providerID,
		Interactive: interactive,
		Messages:    []types.ConversationMessage{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return ChatModel{
		viewport: vp,
		textarea: ta,
		agent:    ag,
		session:  session,
		conversation: []ConversationEntry{
			{
				Type:    "agent",
				Message: "Welcome to the AI Agent Chat! How can I help you today?",
			},
		},
	}
}

func (m ChatModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyCtrlL:
			// Clear conversation and reset session
			m.session = &types.ConversationSession{
				ID:          uuid.New().String(),
				MCPServerID: m.session.MCPServerID,
				ProviderID:  m.session.ProviderID,
				Interactive: m.session.Interactive,
				Messages:    []types.ConversationMessage{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			m.conversation = []ConversationEntry{
				{
					Type:    "agent",
					Message: "Conversation cleared. How can I help you?",
				},
			}
			m.updateViewport()
			return m, nil
		}

		// Handle Enter for sending message (simple approach for TUI)
		if msg.Type == tea.KeyEnter {
			if !m.processing && strings.TrimSpace(m.textarea.Value()) != "" {
				// Send message and start processing
				m.sendMessage()
				return m, m.processMessageCmd()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update viewport and textarea sizes
		headerHeight := 3
		footerHeight := 4
		inputHeight := 3
		
		verticalMargins := headerHeight + footerHeight + inputHeight
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - verticalMargins
		
		m.textarea.SetWidth(msg.Width - 4)
		m.updateViewport()

	case AgentResponseMsg:
		m.processing = false
		
		if msg.Err != nil {
			m.conversation = append(m.conversation, ConversationEntry{
				Type:    "error",
				Message: fmt.Sprintf("❌ Processing Error: %v", msg.Err),
			})
		} else {
			m.conversation = append(m.conversation, ConversationEntry{
				Type:    "agent",
				Message: msg.Message,
				Error:   msg.Error,
			})
		}
		
		m.updateViewport()
		m.viewport.GotoBottom()
		return m, nil

	case ToolCallMsg:
		// Display tool call in conversation
		toolCallText := formatToolCall(msg.ServerName, msg.ToolName, msg.Arguments)
		m.conversation = append(m.conversation, ConversationEntry{
			Type:    "tool",
			Message: toolCallText,
		})
		m.updateViewport()
		return m, nil

	case ToolResultMsg:
		// Display tool result in conversation
		resultText := formatToolResult(msg.Result, msg.Duration)
		m.conversation = append(m.conversation, ConversationEntry{
			Type:    "tool",
			Message: resultText,
		})
		m.updateViewport()
		return m, nil

	case ToolErrorMsg:
		// Display tool error in conversation
		errorText := formatToolError(msg.Error)
		m.conversation = append(m.conversation, ConversationEntry{
			Type:    "error",
			Message: errorText,
		})
		m.updateViewport()
		return m, nil

	case ProgressMsg:
		// Display progress message
		progressText := formatProgress(msg.Message)
		m.conversation = append(m.conversation, ConversationEntry{
			Type:    "progress",
			Message: progressText,
		})
		m.updateViewport()
		return m, nil

	case SummaryMsg:
		// Display execution summary
		summaryText := formatSummary(msg.Summary)
		m.conversation = append(m.conversation, ConversationEntry{
			Type:    "summary",
			Message: summaryText,
		})
		m.updateViewport()
		return m, nil
	}

	// Update textarea
	if !m.processing {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ChatModel) sendMessage() {
	userMessage := strings.TrimSpace(m.textarea.Value())
	if userMessage == "" {
		return
	}

	// Add user message to conversation
	m.conversation = append(m.conversation, ConversationEntry{
		Type:    "user",
		Message: userMessage,
	})

	// Clear textarea
	m.textarea.Reset()
	m.processing = true
	m.updateViewport()
}

func (m ChatModel) processMessageCmd() tea.Cmd {
	if len(m.conversation) == 0 {
		return nil
	}
	
	// Get the last user message
	var lastUserMessage string
	for i := len(m.conversation) - 1; i >= 0; i-- {
		if m.conversation[i].Type == "user" {
			lastUserMessage = m.conversation[i].Message
			break
		}
	}
	
	if lastUserMessage == "" {
		return nil
	}

	return func() tea.Msg {
		// Use a simple non-interactive display for TUI mode to avoid complexity
		// The TUI itself will handle the visual feedback
		display := NewSimpleTUIDisplay()
		
		response, err := m.agent.ProcessConversation(m.session, lastUserMessage, display)
		if err != nil {
			return AgentResponseMsg{Err: err}
		}
		return AgentResponseMsg{
			Message: response.Message,
			Error:   response.Error,
		}
	}
}

func (m *ChatModel) updateViewport() {
	var content strings.Builder
	
	for _, entry := range m.conversation {
		switch entry.Type {
		case "user":
			content.WriteString(userMessageStyle.Render("💬 You: " + entry.Message))
			content.WriteString("\n\n")
		case "agent":
			content.WriteString(agentMessageStyle.Render("🤖 Agent: " + entry.Message))
			if entry.Error != "" {
				content.WriteString("\n")
				content.WriteString(errorStyle.Render("⚠️  Warning: " + entry.Error))
			}
			content.WriteString("\n\n")
		case "error":
			content.WriteString(errorStyle.Render("❌ " + entry.Message))
			content.WriteString("\n\n")
		case "tool":
			// Tool calls and results are pre-formatted
			content.WriteString(entry.Message)
			content.WriteString("\n\n")
		case "progress":
			// Progress messages are pre-formatted
			content.WriteString(entry.Message)
			content.WriteString("\n")
		case "summary":
			// Summary messages are pre-formatted
			content.WriteString(entry.Message)
			content.WriteString("\n\n")
		}
	}

	if m.processing {
		content.WriteString(agentMessageStyle.Render("🔄 Processing your request... Please wait."))
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
}

func (m ChatModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Header
	header := titleStyle.Render("🤖 AI Agent Chat")
	
	// Help text
	help := helpStyle.Render("Enter: Send • Ctrl+L: Clear • Esc: Quit")
	
	// Viewport (conversation history)
	viewportContent := viewportStyle.Render(m.viewport.View())
	
	// Input area
	var inputContent string
	if m.processing {
		inputContent = inputStyle.Render("Processing... Please wait.")
	} else {
		inputContent = inputStyle.Render(m.textarea.View())
	}
	
	// Combine all elements
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		help,
		"",
		viewportContent,
		"",
		inputContent,
	)
}

// StartTUIChat starts the TUI chat interface
func StartTUIChat(ag *agent.Agent, mcpServerID, providerID string, interactive bool) error {
	model := NewChatModel(ag, mcpServerID, providerID, interactive)
	
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	_, err := p.Run()
	return err
}