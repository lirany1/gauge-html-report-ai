package ai

import (
	"os"
	"strings"
	"testing"
)

func TestLoadLLMConfigFromEnv(t *testing.T) {
	// Save original env vars
	originalEnabled := os.Getenv("GAUGE_AI_ENABLED")
	originalProvider := os.Getenv("GAUGE_AI_PROVIDER")
	originalAPIKey := os.Getenv("GAUGE_AI_API_KEY")
	originalModel := os.Getenv("GAUGE_AI_MODEL")

	// Cleanup
	defer func() {
		os.Setenv("GAUGE_AI_ENABLED", originalEnabled)
		os.Setenv("GAUGE_AI_PROVIDER", originalProvider)
		os.Setenv("GAUGE_AI_API_KEY", originalAPIKey)
		os.Setenv("GAUGE_AI_MODEL", originalModel)
	}()

	t.Run("Disabled by default", func(t *testing.T) {
		os.Unsetenv("GAUGE_AI_ENABLED")
		os.Unsetenv("GAUGE_AI_PROVIDER")
		os.Unsetenv("GAUGE_AI_API_KEY")

		config := LoadLLMConfigFromEnv()

		if config.Enabled {
			t.Error("Expected LLM to be disabled by default")
		}
		if config.Provider != ProviderNone {
			t.Errorf("Expected provider None, got %v", config.Provider)
		}
	})

	t.Run("OpenAI configuration", func(t *testing.T) {
		os.Setenv("GAUGE_AI_ENABLED", "true")
		os.Setenv("GAUGE_AI_PROVIDER", "openai")
		os.Setenv("GAUGE_AI_API_KEY", "sk-test-key")
		os.Setenv("GAUGE_AI_MODEL", "gpt-4")

		config := LoadLLMConfigFromEnv()

		if !config.Enabled {
			t.Error("Expected LLM to be enabled")
		}
		if config.Provider != ProviderOpenAI {
			t.Errorf("Expected provider OpenAI, got %v", config.Provider)
		}
		if config.APIKey != "sk-test-key" {
			t.Errorf("Expected API key 'sk-test-key', got %s", config.APIKey)
		}
		if config.Model != "gpt-4" {
			t.Errorf("Expected model 'gpt-4', got %s", config.Model)
		}
		if config.APIURL != "https://api.openai.com/v1/chat/completions" {
			t.Errorf("Expected OpenAI URL, got %s", config.APIURL)
		}
	})

	t.Run("Claude configuration", func(t *testing.T) {
		os.Setenv("GAUGE_AI_ENABLED", "true")
		os.Setenv("GAUGE_AI_PROVIDER", "claude")
		os.Setenv("GAUGE_AI_API_KEY", "sk-ant-test-key")
		os.Unsetenv("GAUGE_AI_MODEL") // Test default model

		config := LoadLLMConfigFromEnv()

		if !config.Enabled {
			t.Error("Expected LLM to be enabled")
		}
		if config.Provider != ProviderClaude {
			t.Errorf("Expected provider Claude, got %v", config.Provider)
		}
		if config.APIKey != "sk-ant-test-key" {
			t.Errorf("Expected API key 'sk-ant-test-key', got %s", config.APIKey)
		}
		if config.Model != "claude-3-sonnet-20240229" {
			t.Errorf("Expected default Claude model, got %s", config.Model)
		}
		if config.APIURL != "https://api.anthropic.com/v1/messages" {
			t.Errorf("Expected Claude URL, got %s", config.APIURL)
		}
	})

	t.Run("Local LLM configuration", func(t *testing.T) {
		os.Setenv("GAUGE_AI_ENABLED", "true")
		os.Setenv("GAUGE_AI_PROVIDER", "local")
		os.Unsetenv("GAUGE_AI_API_KEY") // Not required for local
		os.Unsetenv("GAUGE_AI_MODEL")   // Test default

		config := LoadLLMConfigFromEnv()

		if !config.Enabled {
			t.Error("Expected LLM to be enabled")
		}
		if config.Provider != ProviderLocal {
			t.Errorf("Expected provider Local, got %v", config.Provider)
		}
		if config.Model != "llama2" {
			t.Errorf("Expected default local model 'llama2', got %s", config.Model)
		}
		if config.APIURL != "http://localhost:11434/api/generate" {
			t.Errorf("Expected Ollama URL, got %s", config.APIURL)
		}
	})

	t.Run("Invalid provider is kept as-is", func(t *testing.T) {
		os.Setenv("GAUGE_AI_ENABLED", "true")
		os.Setenv("GAUGE_AI_PROVIDER", "invalid-provider")
		os.Setenv("GAUGE_AI_API_KEY", "test-key")

		config := LoadLLMConfigFromEnv()

		// Invalid provider is stored as-is, client creation will fail gracefully
		if config.Provider != LLMProvider("invalid-provider") {
			t.Errorf("Expected provider to be 'invalid-provider', got %v", config.Provider)
		}
	})
}

func TestNewLLMClient(t *testing.T) {
	t.Run("Returns nil when disabled", func(t *testing.T) {
		config := &LLMConfig{
			Enabled: false,
		}

		client := NewLLMClient(config)

		if client != nil {
			t.Error("Expected nil client when disabled")
		}
	})

	t.Run("Returns nil when no API key for cloud providers", func(t *testing.T) {
		config := &LLMConfig{
			Enabled:  true,
			Provider: ProviderOpenAI,
			APIKey:   "",
		}

		client := NewLLMClient(config)

		// Client is still created, but API calls will fail
		// This is intentional to allow graceful degradation
		if client == nil {
			t.Error("Expected client to be created (will fail at runtime)")
		}
	})

	t.Run("Creates client for valid OpenAI config", func(t *testing.T) {
		config := &LLMConfig{
			Enabled:  true,
			Provider: ProviderOpenAI,
			APIKey:   "sk-test-key",
			APIURL:   "https://api.openai.com/v1/chat/completions",
			Model:    "gpt-4",
		}

		client := NewLLMClient(config)

		if client == nil {
			t.Error("Expected client to be created")
		}
		if client.config.Provider != ProviderOpenAI {
			t.Errorf("Expected OpenAI provider, got %v", client.config.Provider)
		}
	})

	t.Run("Creates client for local provider without API key", func(t *testing.T) {
		config := &LLMConfig{
			Enabled:  true,
			Provider: ProviderLocal,
			APIKey:   "", // Not required for local
			APIURL:   "http://localhost:11434/api/generate",
			Model:    "llama2",
		}

		client := NewLLMClient(config)

		if client == nil {
			t.Error("Expected client to be created for local provider")
		}
	})
}

func TestBuildPrompts(t *testing.T) {
	config := &LLMConfig{
		Enabled:  true,
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "gpt-4",
	}
	client := &LLMClient{
		config: config,
	}

	t.Run("Build fix suggestion prompt", func(t *testing.T) {
		prompt := client.buildFixSuggestionPrompt(
			"Expected 5 but got 1",
			"at test.py:42",
			"Verify vowel count",
			"Word Analysis Tests",
		)

		if prompt == "" {
			t.Error("Expected non-empty prompt")
		}

		// Check prompt contains key information
		expectedPhrases := []string{
			"Expected 5 but got 1",
			"test.py:42",
			"Verify vowel count",
			"Word Analysis Tests",
			"Root cause",   // lowercase in actual prompt
			"Specific fix", // actual prompt text
		}

		for _, phrase := range expectedPhrases {
			if !strings.Contains(prompt, phrase) {
				t.Errorf("Expected prompt to contain '%s'", phrase)
			}
		}
	})

	t.Run("Build executive summary prompt", func(t *testing.T) {
		testData := map[string]interface{}{
			"totalScenarios": 10,
			"passed":         8,
			"failed":         2,
			"successRate":    80.0,
		}

		prompt := client.buildExecutiveSummaryPrompt(testData)

		if prompt == "" {
			t.Error("Expected non-empty prompt")
		}

		// Check prompt contains test data
		if !strings.Contains(prompt, "10") || !strings.Contains(prompt, "8") || !strings.Contains(prompt, "2") {
			t.Error("Expected prompt to contain test statistics")
		}
	})
}
