package callbacks

import (
	"io"
	"log"
	"net/http"
)

const maxBodyBytes = 1 << 20

func Telegram() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		log.Printf("%s", body)
		w.WriteHeader(http.StatusOK)
	})
}
