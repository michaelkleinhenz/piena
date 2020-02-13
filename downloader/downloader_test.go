package downloader

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/michaelkleinhenz/piena/base"
)

func TestDownloader(t *testing.T) {
	// create temp directory
	path, err := ioutil.TempDir("", "piena-")
	assert.NoError(t, err)
	defer os.RemoveAll(path)
	// create directory
	directory := base.AudiobookDirectory{
		ID:      "testDirectory",
		BaseURL: "file:",
		Books: []base.Audiobook{
			base.Audiobook{
				ID:          "testBook",
				Title:       "The Test Book",
				ArchiveFile: "archive.zip",
				Tracks: []base.AudiobookTrack{
					base.AudiobookTrack{Ord: 1, Title: "01", Filename: "01.mp3"},
					base.AudiobookTrack{Ord: 2, Title: "02", Filename: "02.mp3"},
					base.AudiobookTrack{Ord: 3, Title: "03", Filename: "03.mp3"},
					base.AudiobookTrack{Ord: 4, Title: "04", Filename: "04.mp3"},
					base.AudiobookTrack{Ord: 5, Title: "05", Filename: "05.mp3"},
				},
			},
		},
	}
	// create dummy zip file
	var files []string
	for _, entry := range directory.Books[0].Tracks {
		files = append(files, entry.Filename)
	}
	zipfilePath := path + "/" + "archive.zip"
	err = createDummyZipFile(files, zipfilePath)
	assert.NoError(t, err)
	// start testing
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/directory.json" {
            w.WriteHeader(http.StatusOK)
            w.Header().Set("Content-Type", "application/json")
            directoryBytes, _ := json.MarshalIndent(directory, "", " ")
            w.Write(directoryBytes)    
        } else if r.URL.Path == "/archive.zip" {
            w.WriteHeader(http.StatusOK)
            w.Header().Set("Content-Type", "application/octet-stream")
            zipContent, _ := ioutil.ReadFile(zipfilePath)
            w.Write(zipContent)    
        }
	}))
	defer ts.Close()
	directory.BaseURL = ts.URL
	// start test
	downloader, err := NewDownloader(path, ts.URL + "/directory.json")
	assert.NoError(t, err)
	audiobook, err := downloader.GetAudiobook("testBook")
	assert.NoError(t, err)
	assert.NotNil(t, audiobook)
	// check if book is available
	assert.DirExists(t, path + "/" + directory.Books[0].Title)
	for _, entry := range directory.Books[0].Tracks {
			assert.FileExists(t, path + "/" + directory.Books[0].Title + "/" + entry.Filename)
	}
	// download it again
	audiobook, err = downloader.GetAudiobook("testBook")
	assert.NoError(t, err)
	assert.NotNil(t, audiobook)
	// check if book is available
	assert.DirExists(t, path + "/" + directory.Books[0].Title)
	for _, entry := range directory.Books[0].Tracks {
			assert.FileExists(t, path + "/" + directory.Books[0].Title + "/" + entry.Filename)
	}
}

func checkExistence(filepath string) bool {
	if _, err := os.Stat(filepath); err == nil {
		return true
	}
	return false
}

func createDummyZipFile(files []string, zipfile string) error {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for _, file := range files {
		f, err := w.Create(file)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("dummy content"))
		if err != nil {
			return err
		}
	}
	err := w.Close()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(zipfile, buf.Bytes(), 0644)
	if err != nil {
		return err
	}
	return nil
}

