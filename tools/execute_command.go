package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/quniob/shellm/config"

	"golang.org/x/crypto/ssh"
)

type ExecuteCommandArgs struct {
	HostID  string `json:"host_id"`
	Command string `json:"command"`
}

type ExecuteCommand struct {
	HostsData *config.Hosts
}

func (ExecuteCommand) Name() string { return "execute_command" }
func (ExecuteCommand) Description() string {
	return "Executes given command on the specified host. Host ID can be obtained from the get_hosts tool."
}
func (ExecuteCommand) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"host_id": map[string]any{"type": "string"},
			"command": map[string]any{"type": "string"},
		},
		"required": []string{"host_id", "command"},
	}
}
func (e ExecuteCommand) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args ExecuteCommandArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}
	host := e.HostsData.Hosts[args.HostID]
	if host.ID == "" {
		return "", fmt.Errorf("Host with ID '%s' does not exist", args.HostID)
	}
	return e.Run(ctx, host, args.Command)
}

func (e ExecuteCommand) getSSHConfig(secret config.Secret) (*ssh.ClientConfig, error) {
	var authMethod ssh.AuthMethod
	switch secret.Type {
	case "password":
		authMethod = ssh.Password(secret.Password)
	case "keyfile":
		key, err := os.ReadFile(secret.KeyfilePath)
		if err != nil {
			return nil, fmt.Errorf("Unable to read private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("Unable to parse private key: %w", err)
		}
		authMethod = ssh.PublicKeys(signer)
	default:
		return nil, fmt.Errorf("Unsupported authentication type: %s", secret.Type)
	}

	return &ssh.ClientConfig{
		User: secret.User,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}, nil
}

func (e ExecuteCommand) Run(ctx context.Context, host config.Host, command string) (string, error) {
	secret := e.HostsData.Secrets[host.SecretRef]
	config, err := e.getSSHConfig(secret)
	if err != nil {
		return "", err
	}

	addr := fmt.Sprintf("%s:%d", host.Host, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("Failed to dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("Failed to create session: %w", err)
	}
	defer session.Close()

	type result struct {
		output string
		err    error
	}
	resultCh := make(chan result, 1)

	go func() {
		output, err := session.CombinedOutput(command)
		if err != nil {
			resultCh <- result{output: "", err: fmt.Errorf("Failed to run command: %w. Output: %s", err, string(output))}
			return
		}
		resultCh <- result{output: string(output), err: nil}
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGINT)
		return "", ctx.Err()
	case res := <-resultCh:
		return res.output, res.err
	}
}
