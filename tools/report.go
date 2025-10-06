package tools

import (
	"context"
	"encoding/json"
)

type ReportArgs struct {
	Text string `json:"text"`
}
type Report struct{}

func (a Report) Name() string        { return "report" }
func (a Report) Description() string { return "Provides final answer to user" }
func (a Report) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
		"required": []string{"text"},
	}
}

func (a Report) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var ans ReportArgs
	if err := json.Unmarshal(raw, &ans); err != nil {
		return "", err
	}
	return ans.Text, nil
}
