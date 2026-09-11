package pages

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
	"uuid"

	"github.com/paveltessman/yaa/pipelines/shared/ports/history"
)

//go:embed templates/*.html
var templates embed.FS

var listTemplate = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/list.html"))
var detailsTemplate = template.Must(template.ParseFS(templates, "templates/layout.html", "templates/details.html"))

const HistoryPath = "/history/"

const listLimit = 100

const dateFormat = "2006-01-02 15:04:05"

type detailView struct {
	Title string
	Body  string
}

type entryView struct {
	At          string
	Kind        string
	Title       string
	Description string
	Details     []detailView
}

// sessionView holds one record ready for the template. ParentID is empty when
// the pass has no parent.
type sessionView struct {
	SessionID string
	ParentID  string
	Pipeline  string
	Date      string
	Entries   []entryView
}

type listPage struct {
	Title    string
	Sessions []sessionView
}

type detailsPage struct {
	Title   string
	Session sessionView
}

func History(loader history.Loader) http.Handler {
	if loader == nil {
		panic("history loader object is nil")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sessionID := strings.Trim(strings.TrimPrefix(r.URL.Path, HistoryPath), "/")
		if len(sessionID) == 0 {
			showList(r.Context(), w, loader)
			return
		}
		showDetails(r.Context(), w, loader, sessionID)
	})
	return handler
}

func showList(ctx context.Context, w http.ResponseWriter, loader history.Loader) {
	records, err := loader.List(ctx, listLimit)
	if err != nil {
		log.Printf("history page: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sessions := make([]sessionView, 0, len(records))
	for _, record := range records {
		sessions = append(sessions, toView(record))
	}
	render(w, listTemplate, listPage{Title: "sessions", Sessions: sessions})
}

func showDetails(ctx context.Context, w http.ResponseWriter, loader history.Loader, rawID string) {
	sessionID, err := uuid.Parse(rawID)
	if err != nil {
		http.Error(w, "bad session id", http.StatusBadRequest)
		return
	}

	record, err := loader.Get(ctx, sessionID)
	if errors.Is(err, history.ErrNotFound) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("history page: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	render(w, detailsTemplate, detailsPage{Title: "session " + record.Pipeline, Session: toView(record)})
}

// render writes the page into a buffer first, so a broken template gives a 500
// and no half written body.
func render(w http.ResponseWriter, page *template.Template, data any) {
	var body bytes.Buffer
	if err := page.ExecuteTemplate(&body, "layout", data); err != nil {
		log.Printf("history page: can't render: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := body.WriteTo(w); err != nil {
		log.Printf("history page: can't write the answer: %v", err)
	}
}

func toView(record *history.Record) sessionView {
	entries := make([]entryView, 0, len(record.Entries))
	for _, entry := range record.Entries {
		entries = append(entries, toEntryView(entry))
	}

	view := sessionView{
		SessionID: record.SessionID.String(),
		ParentID:  parentOf(record.ParentID),
		Pipeline:  record.Pipeline,
		Date:      record.Date.Format(dateFormat),
		Entries:   entries,
	}
	return view
}

func toEntryView(entry history.Entry) entryView {
	details := make([]detailView, 0, len(entry.Details))
	for _, detail := range entry.Details {
		details = append(details, detailView{Title: detail.Title, Body: prettyBody(detail.Body)})
	}

	view := entryView{
		At:          entry.At.Format(dateFormat),
		Kind:        string(entry.Kind),
		Title:       entry.Title,
		Description: entry.Description,
		Details:     details,
	}
	return view
}

// prettyBody indents a json body over more than one line. A body that is no
// json stays as it is.
func prettyBody(body string) string {
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(body), "", "  "); err != nil {
		return body
	}
	return indented.String()
}

func parentOf(parentID uuid.UUID) string {
	if parentID == uuid.Nil() {
		return ""
	}
	return parentID.String()
}
