package cbhttp

import "context"

func Context(ctx context.Context) RequestOption {
	return func(r *Request) *Request {
		r.Context = ctx
		return r
	}
}
