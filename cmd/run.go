package main

import (
	"context"
	"log"
	"strings"

	ports "github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/platform/api"
	"github.com/paveltessman/yaa/platform/db"
	"github.com/paveltessman/yaa/platform/db/repos/tgupdates"
	"github.com/paveltessman/yaa/platform/llm"
	"github.com/paveltessman/yaa/platform/llm/anthropic"
	"github.com/paveltessman/yaa/platform/llm/ollama"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

func newBackends(s *settings.Settings) map[ports.Model]llm.LLMBackend {
	anthropicBackend := anthropic.NewClient(anthropic.NewSession(s.AnthropicApiKey))
	backends := map[ports.Model]llm.LLMBackend{
		ports.Opus5:   anthropicBackend,
		ports.Sonnet5: anthropicBackend,
		ports.Haiku45: anthropicBackend,
	}

	if len(s.OllamaModels) == 0 {
		return backends
	}

	ollamaBackend := ollama.NewClient(ollama.NewSession(s.OllamaHost))
	for _, model := range s.OllamaModels {
		backends[ports.Model(model)] = ollamaBackend
	}
	log.Printf("Ollama backend ready on %s for %s", s.OllamaHost, strings.Join(s.OllamaModels, ", "))

	return backends
}

func run(ctx context.Context, _ []string) error {
	s := settings.NewSettings()
	log.Println("Settings loaded")

	tgClient := telegram.NewClient(telegram.NewSession(s.TgToken))
	_, err := tgClient.GetMe(ctx)
	if err != nil {
		return err
	}

	pool, err := db.NewPool(ctx, s.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Println("Database pool ready")

	llmService := llm.NewLLMService(newBackends(&s))
	log.Println("LLM service ready")

	deps := api.NewDeps(&s, tgClient, tgupdates.New(pool), llmService)

	if err := api.Serve(ctx, deps); err != nil {
		return err
	}
	return nil
}
