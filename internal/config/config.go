package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config holds runtime configuration. Keys are loaded from, in order of
// precedence: env vars, ~/.config/lean-mac/config.toml. Never logged.
type Config struct {
	GroqKeys []string
	Model    string
}

// Default returns the model defaults; keys empty.
func Default() Config {
	return Config{Model: "llama-3.1-8b-instant"}
}

// Load reads env + config file. Returns a usable Config even if both are empty
// (HasAI() will simply return false).
func Load() Config {
	c := Default()
	// 1. env vars: GROQ_API_KEY, GROQ_API_KEY_2, ..., GROQ_API_KEY_9
	if k := strings.TrimSpace(os.Getenv("GROQ_API_KEY")); k != "" {
		c.GroqKeys = append(c.GroqKeys, k)
	}
	for i := 2; i <= 9; i++ {
		k := strings.TrimSpace(os.Getenv("GROQ_API_KEY_" + itoa(i)))
		if k != "" {
			c.GroqKeys = append(c.GroqKeys, k)
		}
	}
	if m := strings.TrimSpace(os.Getenv("LEAN_MAC_MODEL")); m != "" {
		c.Model = m
	}
	// 2. config file (minimal parser — key=value lines, # for comments)
	if path, err := configPath(); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			c.mergeFile(string(data))
		}
	}
	c.GroqKeys = dedup(c.GroqKeys)
	return c
}

func (c *Config) mergeFile(data string) {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
		switch {
		case k == "model":
			if v != "" {
				c.Model = v
			}
		case strings.HasPrefix(k, "groq_api_key"):
			if v != "" {
				c.GroqKeys = append(c.GroqKeys, v)
			}
		}
	}
}

// HasAI reports whether AI features should be enabled.
func (c Config) HasAI() bool { return len(c.GroqKeys) > 0 }

// configPath returns ~/.config/lean-mac/config.toml.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("no home dir")
	}
	return filepath.Join(home, ".config", "lean-mac", "config.toml"), nil
}

// EnsureConfigDir creates the config directory if missing.
func EnsureConfigDir() (string, error) {
	p, err := configPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", err
	}
	return p, nil
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func itoa(i int) string {
	return string(rune('0' + i))
}
