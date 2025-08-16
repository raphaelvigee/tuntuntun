package tuntunhttp

import (
	"context"
	"net/http"
	"net/url"
)

type Request struct {
	URL     *url.URL
	Headers http.Header
}

type metaKey struct{}

func contextWithMeta(ctx context.Context, r Request) context.Context {
	return context.WithValue(ctx, metaKey{}, r)
}

func RequestFromContext(ctx context.Context) Request {
	return ctx.Value(metaKey{}).(Request)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, _ := url.Parse(r.URL.String())

		ctx := r.Context()
		ctx = contextWithMeta(ctx, Request{
			URL:     u,
			Headers: r.Header.Clone(),
		})
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
