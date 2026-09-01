package network

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ApplicationJson ContentType = "application/json"
)

var _ HTTPRequester = (*HTTPSession)(nil)

const defaultTimeout = 10 * time.Second
const maxBodyBytes = 1024 * 1024 // 1 MB

var ErrHTTPFailed = errors.New("http request failed")

type HTTPSession struct {
	baseURL string
	client  *http.Client
}

func NewHTTPSession(baseURL string) *HTTPSession {
	baseURL = prepareBaseURL(baseURL)
	client := &http.Client{Timeout: defaultTimeout}
	s := &HTTPSession{
		baseURL: baseURL,
		client:  client,
	}
	return s
}

func (s *HTTPSession) GetBytes(path string) ([]byte, error) {
	req, err := s.newRequest(http.MethodGet, path, "", nil)
	if err != nil {
		return nil, err
	}
	return s.do(req)
}

func (s *HTTPSession) PostBytes(path string, contentType ContentType, body []byte) ([]byte, error) {
	req, err := s.newRequest(http.MethodPost, path, contentType, body)
	if err != nil {
		return nil, err
	}
	return s.do(req)
}

func (s *HTTPSession) newRequest(method, path string, contentType ContentType, body []byte) (*http.Request, error) {
	path = preparePath(path)
	u, err := url.JoinPath(s.baseURL, path)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHTTPFailed, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", string(contentType))
	}
	return req, nil
}

func (s *HTTPSession) do(req *http.Request) ([]byte, error) {
	r, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHTTPFailed, err)
	}
	defer func() { _ = r.Body.Close() }()

	if r.StatusCode < 200 || r.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, maxBodyBytes))
		return nil, fmt.Errorf("%w: %s", ErrHTTPFailed, r.Status)
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w: %s", ErrHTTPFailed, err, body[:min(len(body), 100)])
	}
	if int64(len(body)) > maxBodyBytes {
		return nil, fmt.Errorf("%w: body is over max limit (%d)", ErrHTTPFailed, maxBodyBytes)
	}
	return body, nil
}

func prepareBaseURL(raw string) string {
	if raw == "" {
		panic("empty base url is not supported")
	}

	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	if u.Host == "" {
		panic("base url must contain host")
	}
	if u.Scheme == "" {
		panic("base url must contain scheme")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		panic(fmt.Sprintf("unsupported scheme: %q", u.Scheme))
	}
	return strings.TrimSuffix(u.String(), "/")
}

func preparePath(raw string) string {
	segments := strings.Split(raw, "/")
	escaped := make([]string, 0, len(segments))

	for _, seg := range segments {
		if seg == ".." {
			log.Println("'..' was dropped from the path. Is someone trying to hack?")
			continue
		}
		escaped = append(escaped, url.PathEscape(seg))
	}

	path := strings.Join(escaped, "/")
	return strings.TrimPrefix(path, "/")
}
