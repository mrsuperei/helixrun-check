package model

import (
	"fmt"
	"os"
	"strings"

	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
)

// Config describes how to construct a Model instance.
type Config struct {
	Provider  string `json:"provider"`              // e.g. "openai"
	Model     string `json:"model"`                 // e.g. "gpt-4o-mini"
	BaseURL   string `json:"base_url,omitempty"`    // optional override (per-agent)
	APIKeyEnv string `json:"api_key_env,omitempty"` // env var name OR direct key (als hij met sk- begint)
}

// NewModelFromConfig builds a model.Model and a basic GenerationConfig.
func NewModelFromConfig(cfg Config, stream bool) (model.Model, model.GenerationConfig, error) {
	switch cfg.Provider {
	case "openai":
		opts := []openai.Option{}

		// 1) Base URL: JSON override > env var OPENAI_BASE_URL
		baseURL := cfg.BaseURL
		if baseURL == "" {
			if envURL := os.Getenv("OPENAI_BASE_URL"); envURL != "" {
				baseURL = envURL
			}
		}
		if baseURL != "" {
			opts = append(opts, openai.WithBaseURL(baseURL))
		}

		// 2) API key:
		//    - Als cfg.APIKeyEnv met "sk-" begint -> behandel het als directe key.
		//    - Anders: zie het als env-var naam en lees os.Getenv(name).
		var apiKey string
		var apiKeyEnv string

		if cfg.APIKeyEnv != "" {
			if strings.HasPrefix(cfg.APIKeyEnv, "sk-") {
				// directe key in config
				apiKey = cfg.APIKeyEnv
			} else {
				apiKeyEnv = cfg.APIKeyEnv
				apiKey = os.Getenv(apiKeyEnv)
			}
		} else {
			apiKeyEnv = "OPENAI_API_KEY"
			apiKey = os.Getenv(apiKeyEnv)
		}

		if apiKey == "" {
			if apiKeyEnv == "" {
				return nil, model.GenerationConfig{}, fmt.Errorf("missing OpenAI API key (no env and no direct key)")
			}
			return nil, model.GenerationConfig{}, fmt.Errorf("missing OpenAI API key, env %s is empty", apiKeyEnv)
		}

		opts = append(opts, openai.WithAPIKey(apiKey))

		// 3) Model client + generation config
		m := openai.New(cfg.Model, opts...)
		gen := model.GenerationConfig{
			Stream: stream,
		}
		return m, gen, nil

	default:
		return nil, model.GenerationConfig{}, fmt.Errorf("unsupported model provider: %s", cfg.Provider)
	}
}
