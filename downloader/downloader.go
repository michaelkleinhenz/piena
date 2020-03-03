package downloader

import (
	"archive/zip"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/michaelkleinhenz/piena/base"
)

// Downloader is the downloader for audiobooks.
type Downloader struct {
	libraryPath string
	directoryURL string
	directory *base.AudiobookDirectory
}

// NewDownloader returns a new downloader instance.
func NewDownloader(libraryPath string, directoryURL string) (*Downloader, error) {
	downloader := new(Downloader)
	downloader.libraryPath = libraryPath
	downloader.directoryURL = directoryURL
	return downloader, nil
}

// GetAudiobook checks if the audiobook with the given ID is already
// available and (if not) fetches it from the server. Returns nil
// if audiobook is downloaded and available.
func (c *Downloader) GetAudiobook(ID string) (*base.Audiobook, error) {
	// TODO: registry should be cached to make GetID() work offline.
	directoryPath, err := c.downloadFile(c.directoryURL)
	if err != nil && c.directory == nil {
		return nil, err
	}
	directory, err := c.unmarshallDirectoryFile(directoryPath)
	if err != nil {
		return nil, err
	}
	for _, entry := range directory.Books {
		if entry.ID == ID {
			isExisting, err := c.isAudiobookAlreadyExisting(&entry)
			if err != nil {
				return nil, err
			}
			if isExisting {
				return &entry, nil
			}
			err = c.downloadAudiobook(&entry, directory.BaseURL)
			if err != nil {
				return nil, err
			}
			return &entry, nil
		}
	}
	return nil, errors.New("audiobook id not found in directory: " + ID)
}

// GetID retrieves the ID for a given set of artist and title.
func (c *Downloader) GetID(artist string, title string) (string, error) {
	// TODO: read from cache, see above
	directoryPath, err := c.downloadFile(c.directoryURL)
	if err != nil && c.directory == nil {
		return "", err
	}
	directory, err := c.unmarshallDirectoryFile(directoryPath)
	if err != nil {
		return "", err
	}
	for _, entry := range directory.Books {
		if entry.Artist == artist && entry.Title == title {
			return entry.ID, nil
		}
	}
	return "", errors.New("audiobook not found in directory")
}

func (c *Downloader) downloadAudiobook(audiobook *base.Audiobook, baseURL string) error {
	audiobookPath, err := c.getAudiobookPath(audiobook)
	if err != nil {
		return err
	}
	archivePath, err := c.downloadFile(baseURL + "/" + audiobook.ArchiveFile)
	defer c.deleteFile(archivePath)
	if err != nil {
		return err
	}
	err = c.createDirectory(audiobookPath)
	if err != nil {
		return err
	}
	_, err = c.unzip(archivePath, audiobookPath)
	if err != nil {
		return err
	}
	return nil
}

func (c* Downloader) isAudiobookAlreadyExisting(audiobook *base.Audiobook) (bool, error) {
	if audiobook == nil {
		return false, errors.New("given audiobook is nil")
	}
	audiobookPath, err := c.getAudiobookPath(audiobook)
	if err != nil {
		return false, err
	}
	if !c.checkExistence(audiobookPath) {
		return false, nil
	}
	for _, track := range audiobook.Tracks {
		trackFilename, err := c.getTrackPath(audiobook, &track)
		if err != nil {
			return false, err
		}
		if !c.checkExistence(trackFilename) {
			// the directory exists, but one or more of the tracks do 
			// not, remove the entire directory to be robust.
			err = c.deleteDirectory(audiobookPath)
			if err != nil {
				return false, err
			}		
			return false, nil
		}
	}
	return true, nil
}

func (c *Downloader) getAudiobookPath(audiobook *base.Audiobook) (string, error) {
	if audiobook == nil {
		return "", errors.New("given audiobook is nil")
	}
	return c.libraryPath + "/" + audiobook.Title, nil
}

func (c *Downloader) getTrackPath(audiobook *base.Audiobook, track *base.AudiobookTrack) (string, error) {
	if audiobook == nil || track == nil {
		return "", errors.New("given audiobook or track is nil")
	}
	return c.libraryPath + "/" + audiobook.Title + "/" + track.Filename, nil
}

func (c *Downloader) checkExistence(filepath string) bool {
	if _, err := os.Stat(filepath); err == nil {
		return true
	} 
	return false
}

func (c *Downloader) unmarshallDirectoryFile(filepath string) (*base.AudiobookDirectory, error) {
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()
	byteValue, _ := ioutil.ReadAll(jsonFile)
	directory := new(base.AudiobookDirectory)
	json.Unmarshal(byteValue, directory)
	c.directory = directory
	return directory, nil
}

func (c *Downloader) createDirectory(path string) error {
	return os.Mkdir(path, 0755)
}

func (c *Downloader) deleteFile(filepath string) error {
	return os.Remove(filepath)
}

func (c *Downloader) deleteDirectory(dirpath string) error {
	return os.RemoveAll(dirpath)
}

// DownloadFile will download a url to a temporary local file.
func (c *Downloader) downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	out, err := ioutil.TempFile("", "piena")
	if err != nil {
		return "", err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return out.Name(), err
}

func (c *Downloader) unzip(src string, dest string) ([]string, error) {
	var filenames []string
	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()
	for _, f := range r.File {
		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)
		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}
		filenames = append(filenames, fpath)
		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}
		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}
		_, err = io.Copy(outFile, rc)
		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()
		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
