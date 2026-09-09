package main

import (
	"context"
	"log"

	ports "github.com/paveltessman/yaa/pipelines/shared/ports/llm"
	"github.com/paveltessman/yaa/platform/api"
	"github.com/paveltessman/yaa/platform/db"
	"github.com/paveltessman/yaa/platform/db/repos/tgupdates"
	"github.com/paveltessman/yaa/platform/llm"
	"github.com/paveltessman/yaa/platform/llm/anthropic"
	"github.com/paveltessman/yaa/platform/settings"
	"github.com/paveltessman/yaa/platform/telegram"
)

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

	backend := anthropic.NewClient(anthropic.NewSession(s.AnthropicApiKey))
	llmService := llm.NewLLMService(map[ports.Model]llm.LLMBackend{
		ports.Opus5:   backend,
		ports.Sonnet5: backend,
		ports.Haiku45: backend,
	})
	log.Println("LLM service ready")

	deps := api.NewDeps(&s, tgClient, tgupdates.New(pool), llmService)

	if err := api.Serve(ctx, deps); err != nil {
		return err
	}
	return nil
}
