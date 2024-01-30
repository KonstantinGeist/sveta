package common

import (
	"io"
	"net/http"
	"os"
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

// DownloadFromURL downloads the content from the given URL and saves at `localPath`.
func DownloadFromURL(url, localPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	_, err = io.Copy(out, resp.Body)
	return err
}
