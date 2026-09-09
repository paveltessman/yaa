package settings

import (
	"slices"
	"testing"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("TG_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Setenv("PUBLIC_HTTP_HOST", "https://example.org")
}

func TestOllamaModels(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want []string
	}{
		"unset":         {"", nil},
		"one model":     {"qwen3:8b", []string{"qwen3:8b"}},
		"two models":    {"qwen3:8b,gemma3:12b", []string{"qwen3:8b", "gemma3:12b"}},
		"spaces":        {" qwen3:8b , gemma3:12b ", []string{"qwen3:8b", "gemma3:12b"}},
		"empty entries": {",qwen3:8b,,", []string{"qwen3:8b"}},
		"commas only":   {",,", nil},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			setRequired(t)
			t.Setenv("OLLAMA_MODELS", key.raw)

			got := NewSettings().OllamaModels

			if !slices.Equal(got, key.want) {
				t.Errorf("want=%v, got=%v", key.want, got)
			}
		})
	}
}

func TestOllamaHost(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want string
	}{
		"unset":  {"", defaultOllamaHost},
		"docker": {"http://host.docker.internal:11434", "http://host.docker.internal:11434"},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			setRequired(t)
			t.Setenv("OLLAMA_HOST", key.raw)

			got := NewSettings().OllamaHost

			if got != key.want {
				t.Errorf("want=%q, got=%q", key.want, got)
			}
		})
	}
}

func TestGeminiApiKey(t *testing.T) {
	cases := map[string]struct {
		raw  string
		want string
	}{
		"unset":     {"", ""},
		"plain key": {"AIza-123", "AIza-123"},
	}
	for name, key := range cases {
		t.Run(name, func(t *testing.T) {
			setRequired(t)
			t.Setenv("GEMINI_API_KEY", key.raw)

			got := NewSettings().GeminiApiKey

			if got != key.want {
				t.Errorf("want=%q, got=%q", key.want, got)
			}
		})
	}
}
