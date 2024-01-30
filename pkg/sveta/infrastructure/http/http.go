package http

import (
	"io"
	"net/http"
)

func ReadAllFromURL(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(res.Body)
	defer func() {
		_ = res.Body.Close()
	}()
	if err != nil {
		return "", err
	}
	return string(content), nil
}
