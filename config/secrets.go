package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

type Secret struct {
	ID             string `validate:"required" yaml:"id"`
	Type           string `validate:"required,oneof=keyfile password" yaml:"type"`
	User           string `validate:"required" yaml:"user"`
	KeyfilePath    string `validate:"required_if=Type keyfile" yaml:"filepath"`
	Password       string `validate:"excluded_with=PasswordEnvKey" yaml:"password"`
	PasswordEnvKey string `validate:"excluded_with=Password" yaml:"passwordEnvKey"`
}

func secretPasswordValidation(sl validator.StructLevel) {
	secret := sl.Current().Interface().(Secret)

	if secret.Type == "password" {
		if secret.Password == "" && secret.PasswordEnvKey == "" {
			sl.ReportError(secret.Password, "Password", "Password", "required_oneof", "either Password or PasswordEnvKey must be set")
		}
	}
}

func LoadSecrets(path string) (map[string]Secret, error) {
	secrets := make(map[string]Secret, 0)
	var data []Secret

	v := validator.New()
	v.RegisterStructValidation(secretPasswordValidation, Secret{})

	fileContent, err := os.ReadFile(path)
	if err != nil {
		return secrets, err
	}

	if err := yaml.Unmarshal(fileContent, &data); err != nil {
		return secrets, err
	}

	for _, secret := range data {
		if err := v.Struct(secret); err != nil {
			if validationErrors, ok := err.(validator.ValidationErrors); ok {
				fieldError := validationErrors[0]
				return secrets, fmt.Errorf(
					"secret '%s': validation failed for field '%s' on tag '%s'",
					secret.ID,
					fieldError.Field(),
					fieldError.Tag(),
				)
			}
			return secrets, err
		}
		secrets[secret.ID] = secret
	}

	return secrets, nil
}
