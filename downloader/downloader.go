package downloader

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	tempDir string
	directory *base.AudiobookDirectory
}

// NewDownloader returns a new downloader instance.
func NewDownloader(libraryPath string, directoryURL string) (*Downloader, error) {
	downloader := new(Downloader)
	downloader.libraryPath = libraryPath
	downloader.directoryURL = directoryURL
	var err error
	downloader.tempDir, err = ioutil.TempDir("", "piena")
	if err != nil {
		return nil, err
	}
	return downloader, nil
}

// GetAudiobook checks if the audiobook with the given ID is already
// available and (if not) fetches it from the server. Returns nil
// if audiobook is downloaded and available.
func (c *Downloader) GetAudiobook(ID string) (*base.Audiobook, bool, error) {
	log.Printf("[downloader] retrieving audiobook %s", ID)
	directoryPath, err := c.downloadFile(c.directoryURL)
	if err != nil && c.directory == nil {
		return nil, false, err
	}
	directory, err := c.unmarshallDirectoryFile(directoryPath)
	if err != nil {
		return nil, false, err
	}
	for _, entry := range directory.Books {
		if strings.HasPrefix(ID, entry.ID) {
			isExisting, err := c.isAudiobookAlreadyExisting(&entry)
			if err != nil {
				return nil, false, err
			}
			if isExisting {
				return &entry, true, nil
			}
			err = c.downloadAudiobook(&entry, directory.BaseURL)
			if err != nil {
				return nil, false, err
			}
			return &entry, false, nil
		}
	}
	return nil, false, errors.New("audiobook id not found in directory: " + ID)
}

// GetID retrieves the ID for a given set of artist and title.
func (c *Downloader) GetID(artist string, title string) (string, error) {
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
			log.Printf("[downloader] retrieving id for audiobook %s %s: %s", artist, title, entry.ID)
			return entry.ID, nil
		}
	}
	log.Printf("[downloader] retrieving id for audiobook failed: %s %s", artist, title)
	return "", errors.New("audiobook not found in directory")
}

func (c *Downloader) downloadAudiobook(audiobook *base.Audiobook, baseURL string) error {
	log.Printf("[downloader] downloading %s", audiobook.ID)
	audiobookPath, err := c.getAudiobookPath(audiobook)
	if err != nil {
		return err
	}
	archivePath, err := c.downloadFile(baseURL + audiobook.ArchiveFile)
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
	log.Printf("[downloader] checking if audiobook already exists: %s", audiobook.ID)
	if audiobook == nil {
		return false, errors.New("given audiobook is nil")
	}
	audiobookPath, err := c.getAudiobookPath(audiobook)
	if err != nil {
		return false, err
	}
	if !c.checkExistence(audiobookPath) {
		log.Printf("[downloader] audiobook does not exist: %s", audiobook.ID)
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
			log.Printf("[downloader] audiobook has damaged download, will be refetched: %s", audiobook.ID)
			return false, nil
		}
	}
	log.Printf("[downloader] audiobook already exists: %s", audiobook.ID)
	return true, nil
}

func (c *Downloader) getAudiobookPath(audiobook *base.Audiobook) (string, error) {
	if audiobook == nil {
		return "", errors.New("given audiobook is nil")
	}
	path := c.libraryPath + "/" + audiobook.Artist + "/" + audiobook.Title
	log.Printf("[downloader] returning audiobook path for audiobook %s: %s", audiobook.ID, path)
	return path, nil
}

func (c *Downloader) getTrackPath(audiobook *base.Audiobook, track *base.AudiobookTrack) (string, error) {
	if audiobook == nil || track == nil {
		return "", errors.New("given audiobook or track is nil")
	}
	path := c.libraryPath + "/" + audiobook.Artist + "/" + audiobook.Title + "/" + track.Filename
	return path, nil
}

func (c *Downloader) checkExistence(filepath string) bool {
	if _, err := os.Stat(filepath); err == nil {
		return true
	} 
	return false
}

func (c *Downloader) unmarshallDirectoryFile(filepath string) (*base.AudiobookDirectory, error) {
	log.Printf("[downloader] unmarshalling %s", filepath)
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
	log.Printf("[downloader] creating directory %s", path)
	return os.MkdirAll(path, 0755)
}

func (c *Downloader) deleteFile(filepath string) error {
	log.Printf("[downloader] deleting file %s", filepath)
	return os.Remove(filepath)
}

func (c *Downloader) deleteDirectory(dirpath string) error {
	log.Printf("[downloader] deleting directory %s", dirpath)
	return os.RemoveAll(dirpath)
}

func (c *Downloader) hashURL(url string) string {
	h := sha1.New()
	h.Write([]byte(url))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

// DownloadFile will download a url to a temporary local file.
func (c *Downloader) downloadFile(url string) (string, error) {
	hashedFilename := c.hashURL(url)
	tmpfn := filepath.Join(c.tempDir, hashedFilename)
	log.Printf("[downloader] downloading from %s", url)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(os.Getenv("PIENA_USER"), os.Getenv("PIENA_PASS"))
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		// error, we try to get a cached version
		log.Printf("[downloader] downloading %s failed, trying to use a cached version of the file", url)
		if c.checkExistence(tmpfn) {
			log.Printf("[downloader] returning cached version for %s found at %s", url, tmpfn)
			return tmpfn, nil
		}
		return "", err
	}
	defer resp.Body.Close()
	// write file to temp file that has the url hash as filename
	log.Printf("[downloader] downloading to %s", tmpfn)
	out, err := os.Create(tmpfn)
	if err != nil {
		return "", err
	}	
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return out.Name(), err
}

func (c *Downloader) unzip(src string, dest string) ([]string, error) {
	log.Printf("[downloader] unzipping file %s to %s", src, dest)
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
	log.Printf("[downloader] unzipping of %s done", src)
	return filenames, nil
}
