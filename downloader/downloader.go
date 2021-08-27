package downloader

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filePath, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(resp)
		fmt.Println(err.Error())
		return err
	}
	defer resp.Body.Close()

	// Create the file
	if er := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); er != nil {
		return er
	}
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)

	return err
}
