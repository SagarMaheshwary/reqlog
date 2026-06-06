package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

type SSH struct {
	Defaults Defaults        `yaml:"defaults"`
	Hosts    map[string]Host `yaml:"hosts"`
}

type Defaults struct {
	Key     string        `yaml:"key"`
	Timeout time.Duration `yaml:"timeout"`
}

type Host struct {
	Host    string        `yaml:"host"`
	User    string        `yaml:"user"`
	Port    int           `yaml:"port"`
	Key     string        `yaml:"key"`
	Timeout time.Duration `yaml:"timeout"`
}

const (
	sshDefaultPort    = 22
	sshDefaultTimeout = 30 * time.Second
)

func GetConfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}

	userConfig, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get user config dir: %w", err)
	}

	return filepath.Join(userConfig, "reqlog", "config.yaml"), nil
}

func NewSSH(path string) (*SSH, error) {
	path, err := GetConfigPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(
				"SSH config not found: %s\nCreate one or pass --config <path>",
				path,
			)
		} else {
			return nil, fmt.Errorf("read ssh config %q: %w", path, err)
		}
	}

	cfg, err := ParseSSH(data)
	if err != nil {
		return nil, fmt.Errorf("parse ssh config %q: %w", path, err)
	}

	return cfg, nil
}

func ParseSSH(data []byte) (*SSH, error) {
	var cfg SSH

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal yaml: %w", err)
	}

	cfg.assignDefaults()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func (c *SSH) validate() error {
	if len(c.Hosts) == 0 {
		return errors.New("ssh config: no hosts defined")
	}

	for alias, h := range c.Hosts {
		if h.Host == "" {
			return fmt.Errorf("host %q: missing host", alias)
		}
		if h.User == "" {
			return fmt.Errorf("host %q: missing user", alias)
		}
		if h.Key == "" {
			return fmt.Errorf("host %q: missing key", alias)
		}
	}

	return nil
}

func (c *SSH) assignDefaults() {
	for k, v := range c.Hosts {
		if v.Port == 0 {
			v.Port = sshDefaultPort
		}

		if v.Key == "" {
			v.Key = c.Defaults.Key
		}
		v.Key = expandHomeDir(v.Key)

		if v.Timeout == 0 {
			if c.Defaults.Timeout != 0 {
				v.Timeout = c.Defaults.Timeout
			} else {
				v.Timeout = sshDefaultTimeout
			}
		}
		c.Hosts[k] = v
	}
}

func expandHomeDir(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}

	return path
}
