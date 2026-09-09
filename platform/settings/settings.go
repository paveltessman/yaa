package settings

import (
	"net/url"
	"os"
	"strings"
)

const (
	defaultApiAddr     = "127.0.0.1:8080"
	defaultDatabaseURL = "postgres://yaa:yaa@localhost:5433/yaa?sslmode=disable"
	defaultOllamaHost  = "http://localhost:11434"
)

type Settings struct {
	TgToken         string
	AnthropicApiKey string
	OllamaHost      string
	OllamaModels    []string
	PublicHost      string
	ApiAddr         string
	DatabaseURL     string
}

func DatabaseURL() string {
	url := os.Getenv("DATABASE_URL")
	if len(url) == 0 {
		url = defaultDatabaseURL
	}
	return url
}

// ollamaModels reads the local models as a comma separated list. An empty
// list leaves the ollama backend out.
func ollamaModels() []string {
	raw := strings.Split(os.Getenv("OLLAMA_MODELS"), ",")
	models := make([]string, 0, len(raw))

	for _, name := range raw {
		name = strings.TrimSpace(name)
		if len(name) == 0 {
			continue
		}
		models = append(models, name)
	}
	return models
}

func NewSettings() Settings {
	TgToken := os.Getenv("TG_TOKEN")

	if len(TgToken) == 0 {
		panic("TG_TOKEN is not set")
	}

	AnthropicApiKey := os.Getenv("ANTHROPIC_API_KEY")
	if len(AnthropicApiKey) == 0 {
		panic("ANTHROPIC_API_KEY is not set")
	}

	OllamaHost := os.Getenv("OLLAMA_HOST")
	if len(OllamaHost) == 0 {
		OllamaHost = defaultOllamaHost
	}

	ApiAddr := os.Getenv("API_ADDR")
	if len(ApiAddr) == 0 {
		ApiAddr = defaultApiAddr
	}

	PublicHost := os.Getenv("PUBLIC_HTTP_HOST")
	if _, err := url.Parse(PublicHost); err != nil {
		panic(err)
	}

	settings := Settings{
		TgToken:         TgToken,
		AnthropicApiKey: AnthropicApiKey,
		OllamaHost:      OllamaHost,
		OllamaModels:    ollamaModels(),
		PublicHost:      PublicHost,
		ApiAddr:         ApiAddr,
		DatabaseURL:     DatabaseURL(),
	}
	return settings
}
