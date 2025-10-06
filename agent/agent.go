package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/quniob/shellm/config"
	"github.com/quniob/shellm/tools"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

const systemPrompt = `You are a ReAct agent called "SheLLM" whose goal is to help user to control his SSH hosts. You have 10+ years of experience in Linux administration and DevOps.

Loop (strict):
1) Thought: 1–2 short sentences (high-level, factual, no speculation). State only the immediate next step and the name of the tool you will call. Do NOT invent facts, credentials, or outcomes in the Thought.
2) Action: call exactly ONE tool with a single JSON argument (the tool invocation will be produced by the assistant).
3) Observation: process the tool's output and continue the loop.

Rules:
- Use ONLY ONE tool per Action.
- If a tool returns an error or an unknown tool is requested, report it in the Observation and continue.
- When you want to finish, call the tool named "report" with JSON following its schema; do not output the report in plain text.
- Keep Thoughts concise and actionable.
- If user dont ask you a task - just answer him with report tool
- Use markdown syntax for answer provided to "report" tool`

type ThoughtMsg struct{ Content string }
type ToolCallMsg struct{ Content string }
type ToolResultMsg struct{ Content string }
type FinalResultMsg struct{ Content string }
type TokenUsageMsg struct{ Tokens int }
type ErrMsg struct{ Err error }

type UsageStats struct {
	tokenUsage int
}

type Agent struct {
	config        *config.Config
	client        *openai.Client
	memory        []openai.ChatCompletionMessageParamUnion
	toolsRegistry *tools.Registry
	stats         UsageStats
}

func NewAgent(tr *tools.Registry, cfg *config.Config) *Agent {
	client := openai.NewClient(
		option.WithAPIKey(cfg.ApiKey),
		option.WithBaseURL(cfg.ApiBaseUrl),
	)
	memory := make([]openai.ChatCompletionMessageParamUnion, 0)
	memory = append(memory, openai.SystemMessage(systemPrompt))
	return &Agent{
		config:        cfg,
		toolsRegistry: tr,
		client:        &client,
		memory:        memory,
		stats:         UsageStats{},
	}
}

func (a *Agent) GetStats() UsageStats {
	return a.stats
}

func (a *Agent) completion() (*openai.ChatCompletion, error) {
	chatCompletion, err := a.client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Messages: a.memory,
		Model:    a.config.ApiModel,
		Tools:    a.toolsRegistry.Tools(),
	})
	return chatCompletion, err
}

func (a *Agent) Start(ctx context.Context, userMessage string, msgCh chan<- tea.Msg) {
	a.memory = append(a.memory, openai.UserMessage(userMessage))

	for iter := 0; iter < a.config.LLMMaxIterations; iter++ {
		completion, err := a.completion()
		if err != nil {
			log.Printf("completion error: %v", err)
			msgCh <- ErrMsg{Err: err}
			return
		}
		if completion == nil || len(completion.Choices) == 0 {
			log.Println("empty completion / no choices")
			msgCh <- ErrMsg{Err: fmt.Errorf("empty completion")}
			return
		}

		a.stats.tokenUsage += int(completion.Usage.TotalTokens)
		msgCh <- TokenUsageMsg{Tokens: a.stats.tokenUsage}

		choice := completion.Choices[0]
		assistantMsg := choice.Message

		if assistantMsg.Content != "" || len(assistantMsg.ToolCalls) > 0 {
			msgCh <- ThoughtMsg{Content: assistantMsg.Content}
			a.memory = append(a.memory, openai.AssistantMessage(assistantMsg.Content))
		}

		if len(assistantMsg.ToolCalls) > 0 {
			toolCall := assistantMsg.ToolCalls[0]
			toolName := toolCall.Function.Name
			toolArgs := toolCall.Function.Arguments

			msgCh <- ToolCallMsg{Content: fmt.Sprintf("%s(%s)", toolName, toolArgs)}

			tool, ok := a.toolsRegistry.Get(toolName)
			if !ok {
				errMsg := fmt.Sprintf("unknown tool: %s", toolName)
				a.memory = append(a.memory, openai.ToolMessage(errMsg, toolCall.ID))
				msgCh <- ToolResultMsg{Content: errMsg}
				continue
			}

			resp, err := tool.Call(ctx, json.RawMessage(toolArgs))
			if err != nil {
				errMsg := fmt.Sprintf("tool error: %v", err)
				a.memory = append(a.memory, openai.ToolMessage(errMsg, toolCall.ID))
				msgCh <- ToolResultMsg{Content: errMsg}
				continue
			}

			msgCh <- ToolResultMsg{Content: resp}
			a.memory = append(a.memory, openai.ToolMessage(resp, toolCall.ID))

			if toolName == "report" {
				msgCh <- FinalResultMsg{Content: resp}
				a.memory = append(a.memory, openai.AssistantMessage(resp))
				return
			}
			continue
		}
	}

	msgCh <- FinalResultMsg{Content: "failed: max iterations reached"}
}
