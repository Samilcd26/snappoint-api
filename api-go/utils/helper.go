package utils

import (
	"bytes"
	"io"
	"net/http"
)

// Body'yi oku ve geri yükle
func ReadBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	// Body'yi oku
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	// Okuduktan sonra body'yi geri koy
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Body içeriğini string olarak döndür
	return string(bodyBytes), nil
}
