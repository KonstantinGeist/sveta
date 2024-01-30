package common

import (
	"io"
	"net/http"
)

func ReadAllFromURL(url string) ([]byte, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	content, err := io.ReadAll(res.Body)
	defer func() {
		_ = res.Body.Close()
	}()
	if err != nil {
		return nil, err
	}
	return content, nil
}
