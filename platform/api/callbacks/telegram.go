package callbacks

import (
	"context"
	"io"
	"log"
	"net/http"

	pipelines "github.com/paveltessman/yaa/pipelines/shared"
	"github.com/paveltessman/yaa/pipelines/telegram/updates"
	"github.com/paveltessman/yaa/pipelines/telegram/updates/session"
)

const maxBodyBytes = 1 << 20

const TgWebhookPath = "/v1/callbacks/telegram"

func Telegram(
	runner pipelines.Runner[*session.Session],
	chain []pipelines.Handler[*session.Session],
) http.Handler {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			log.Printf("read telegram callback body failed: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

		session := updates.NewSession(body)
		err = runner(context.TODO(), session, chain)
		if err != nil {
			log.Println(err)
		}
	})
	return handler
}
