// Package network provides code for outgoing network requests.
package network

type ContentType string

type HTTPRequester interface {
	GetBytes(path string) ([]byte, error)
	PostBytes(path string, contentType ContentType, body []byte) ([]byte, error)
}
