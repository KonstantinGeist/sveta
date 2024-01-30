package common

import (
	"io"
	"net/http"
)

// ReadAllFromURL reads all content from the URL.
// TODO Unsafe if the URL is a dynamic page which infinitely streams output -- we can crash with an OOM in that case.
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
