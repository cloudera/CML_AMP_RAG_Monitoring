package lhttptest

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/textproto"
	"pgregory.net/rapid"
)

func MethodGenerator() *rapid.Generator[string] {
	return rapid.OneOf(
		rapid.Just(http.MethodGet),
		rapid.Just(http.MethodPost),
		rapid.Just(http.MethodPut),
		rapid.Just(http.MethodDelete),
	)
}

func UrlGenerator() *rapid.Generator[string] {
	return rapid.StringMatching(`(http://|https://)[a-z]+[a-z0-9-]*(:[0-9]+)?(/[a-z0-9-]+)*/?`)
}

func UrlSegmentGenerator() *rapid.Generator[string] {
	return rapid.StringMatching(`([a-z0-9-]+)*`)
}

func CodeGenerator() *rapid.Generator[int] {
	return rapid.IntRange(200, 599)
}

func HeadersGenerator() *rapid.Generator[http.Header] {
	return rapid.Map(
		rapid.MapOf(
			rapid.Map(
				rapid.StringMatching(`\w+`),
				func(s string) string {
					return textproto.CanonicalMIMEHeaderKey(s)
				},
			),
			rapid.SliceOf(rapid.String()),
		),
		func(v map[string][]string) http.Header { return v })
}

func CheckHeaders(t assert.TestingT, ref, other http.Header) {
	for k, vals := range ref {
		otherVals := other.Values(k)
		assert.Subsetf(t, vals, otherVals, "values don't match for key %s", k)
		assert.Subsetf(t, otherVals, vals, "values don't match for key %s", k)
		assert.NotRegexp(t, "_", k, "header keys can't have underscore")
	}
}
