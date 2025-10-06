package tools

import (
	"context"
	"encoding/json"

	"github.com/quniob/shellm/config"
)

type GetHostsArgs struct{}

type GetHosts struct {
	HostsData *config.Hosts
}

func (GetHosts) Name() string { return "get_hosts" }
func (GetHosts) Description() string {
	return "Gets a list of available hosts from the inventory. Returns a JSON array of host information - ID, Host, Port, Description and tags"
}
func (GetHosts) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

type HostInfo struct {
	ID          string   `json:"id"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

func (h GetHosts) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	sanitizedHosts := make([]HostInfo, len(h.HostsData.Hosts))
	for _, h := range h.HostsData.Hosts {
		sanitizedHosts = append(sanitizedHosts, HostInfo{
			ID:          h.ID,
			Host:        h.Host,
			Port:        h.Port,
			Description: h.Description,
			Tags:        h.Tags,
		})
	}

	out, err := json.MarshalIndent(sanitizedHosts, "", "  ")
	if err != nil {
		return "", err
	}

	return string(out), nil
}
