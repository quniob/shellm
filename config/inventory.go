package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Host struct {
	ID          string   `yaml:"id" validate:"required,alphanumeric"`
	Host        string   `yaml:"host" validate:"hostname,required"`
	Port        int      `yaml:"port" validate:"hostname_port"`
	SecretRef   string   `yaml:"secretRef" validate:"alphanumeric"`
	Description string   `yaml:"description" validate:"alphanumeric,omitempty"`
	Tags        []string `yaml:"tags"`
}

type Hosts struct {
	Hosts   map[string]Host `yaml:"hosts"`
	Secrets map[string]Secret
}

func LoadHosts(inventoryPath string, secretsPath string) (Hosts, error) {
	hostsData := []Host{}
	hosts := Hosts{
		Hosts:   make(map[string]Host),
		Secrets: make(map[string]Secret),
	}

	fileContent, err := os.ReadFile(inventoryPath)
	if err != nil {
		return hosts, err
	}

	err = yaml.Unmarshal(fileContent, &hostsData)
	if err != nil {
		return hosts, err
	}

	secrets, err := LoadSecrets(secretsPath)
	if err != nil {
		return hosts, err
	}

	hosts.Secrets = secrets
	for _, host := range hostsData {
		hosts.Hosts[host.ID] = host
	}

	return hosts, nil
}

func (h *Hosts) List() []string {
	list := make([]string, len(h.Hosts))
	for _, host := range h.Hosts {
		list = append(list, host.Host)
	}
	return list
}
