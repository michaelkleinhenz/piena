package uploader

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	id3 "github.com/mikkyang/id3-go"
	"github.com/stretchr/testify/assert"
)

const (
	numTracks = 5
	exampleMP3Path = "../example.mp3"
)

func TestDownloader(t *testing.T) {
	// create temp directory
	path, err := ioutil.TempDir("", "piena-")
	assert.NoError(t, err)
	defer os.RemoveAll(path)
	uploader, err := NewUploader()
	assert.NoError(t, err)
	exampleMP3Info, err := os.Stat(exampleMP3Path)
	assert.NoError(t, err)
	t.Run("tagging files", func(t *testing.T) {
		// create example source dir
		fileList, err := createSourceMP3Dir(path)
		defer os.RemoveAll(path)
		assert.NoError(t, err)
		packageFiles, err := uploader.TagRenameFiles(path, fileList, "Example Artist", "Example Album Title")
		assert.NoError(t, err)
		for i, packageFile := range(packageFiles) {
			destFile := uploader.tempDir + "/" + packageFile
			// check if file exists
			_, err = os.Stat(destFile)
			assert.NoError(t, err)
			// check if file has correct id3 tags
			mp3File, err := id3.Open(destFile)
			assert.NoError(t, err)
			assert.Equal(t, "Example Artist", mp3File.Artist())
			assert.Equal(t, "Example Album Title", mp3File.Album())
			expectedTitle := fmt.Sprintf("%02d", i)
			assert.Equal(t, expectedTitle, mp3File.Title())
			err = mp3File.Close()
			assert.NoError(t, err)
		}
	})
	t.Run("creating package file", func(t *testing.T) {
		// create example source dir
		fileList, err := createSourceMP3Dir(path)
		defer os.RemoveAll(path)
		assert.NoError(t, err)
		packageFiles, err := uploader.TagRenameFiles(path, fileList, "Example Artist", "Example Album Title")
		assert.NoError(t, err)
		packageFile, err := uploader.PackageFiles(packageFiles, "Example Artist", "Example Album Title")
		assert.NoError(t, err)
		// check if package file exists
		_, err = os.Stat(packageFile)
		assert.NoError(t, err)
		// check if package file is zip and hat the source files
		zip, err := zip.OpenReader(packageFile)
		assert.NoError(t, err)
		defer zip.Close()
		for _, zf := range(zip.File) {
			assert.True(t, contains(packageFiles, zf.Name))
			assert.Equal(t, zf.FileInfo().Size(), exampleMP3Info.Size)
		}
		// check if original files are removed
		for _, pf := range(packageFiles) {
			_, err = os.Stat(pf)
			assert.Error(t, err)
		}
	})	
}

func createSourceMP3Dir(basepath string) ([]string, error) {
	fileList := []string{}
	for i := 1; i <= numTracks; i++ {
		trackfile := "track-" + strconv.Itoa(i) + ".mp3"
		copyFile(exampleMP3Path, basepath + "/" + trackfile)
		fileList = append(fileList, trackfile)
	}
	return fileList, nil
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("[uploader] %s is not a regular file", src)
	}
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func contains(s []string, e string) bool {
	for _, a := range s {
			if a == e {
					return true
			}
	}
	return false
}
