package network

import (
	"context"

	"github.com/paveltessman/yaa/platform/network"
)

type FakeRequester struct {
	Resp []byte
	Err  error

	Calls   int
	GotCtx  context.Context
	GotPath string
	GotType network.ContentType
	GotBody []byte
}

var _ network.HTTPRequester = (*FakeRequester)(nil)

func (f *FakeRequester) GetBytes(ctx context.Context, path string) ([]byte, error) {
	f.Calls++
	f.GotCtx = ctx
	f.GotPath = path
	return f.Resp, f.Err
}

func (f *FakeRequester) PostBytes(ctx context.Context, path string, contentType network.ContentType, body []byte) ([]byte, error) {
	f.Calls++
	f.GotCtx = ctx
	f.GotPath = path
	f.GotType = contentType
	f.GotBody = body
	return f.Resp, f.Err
}
