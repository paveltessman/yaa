// Package network provides code for outgoing network requests.
package network

import "context"

type ContentType string

type HTTPRequester interface {
	GetBytes(ctx context.Context, path string) ([]byte, error)
	PostBytes(ctx context.Context, path string, contentType ContentType, body []byte) ([]byte, error)
}
