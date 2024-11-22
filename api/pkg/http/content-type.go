package lhttp

import (
	"mime"
	"net/http"
	"path/filepath"
)

func InferContentType(name string, content []byte) string {
	ctype := mime.TypeByExtension(filepath.Ext(name))
	if ctype == "" {
		ctype = http.DetectContentType(content)
	}
	return ctype
}
