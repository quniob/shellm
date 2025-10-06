package tools

import (
	"context"
	"encoding/json"
	"os/exec"
)

type PingArgs struct {
	Target string `json:"target"`
}
type Ping struct{}

func (Ping) Name() string        { return "ping" }
func (Ping) Description() string { return "Pings given ipv4 target" }
func (Ping) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"target": map[string]any{"type": "string"},
		},
		"required": []string{"target"},
	}
}
func (Ping) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var a PingArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return "", err
	}
	cmd := exec.Command("ping", a.Target, "-c 5")
	out, err := cmd.CombinedOutput()
	return string(out), err
}
