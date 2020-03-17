package uploader

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	id3 "github.com/mikkyang/id3-go"
	/*
		"github.com/michaelkleinhenz/piena/base"
		"github.com/michaelkleinhenz/piena/downloader"
	*/)

// Uploader is the downloader for audiobooks.
type Uploader struct {
	tempDir string
}

// NewUploader returns a new uploader instance.
func NewUploader() (*Uploader, error) {
	uploader := new(Uploader)
	var err error
	uploader.tempDir, err = ioutil.TempDir("", "piena-upload")
	if err != nil {
		return nil, err
	}
	return uploader, nil
}

func (u *Uploader) TagRenameFiles(uploadFileDir string, uploadFiles []string, uploadArtist string, uploadTitle string) ([]string, error) {
	packageFiles := []string{}
	for idx, filename := range(uploadFiles) {
		srcFile := uploadFileDir + "/" + filename
		destTitle := fmt.Sprintf("%02d", idx)
		destFilename := destTitle + ".mp3"
		destFilePath := u.tempDir + "/" + destFilename
		log.Printf("[uploader] copy/tagging file %s to destination %s", srcFile, destFilePath)
		// copy file.
		u.copyFile(srcFile, destFilePath)
		// tag file.
		mp3File, err := id3.Open(destFilePath)
		if err != nil {
			return nil, err
		}
		mp3File.SetArtist(uploadArtist)
		mp3File.SetAlbum(uploadTitle)
		mp3File.SetTitle(destTitle)
		err = mp3File.Close()
		if err != nil {
			return nil, err
		}
		packageFiles = append(packageFiles, destFilename)
	}
	return packageFiles, nil
}

func (u *Uploader) PackageFiles(packageFiles []string, artist string, title string) (string, error) {
	// TODO: escape strings here
	packageFilename := u.tempDir + "/" + artist + " - " + title + ".zip"
	log.Printf("[uploader] creating package file %s from files %s", packageFilename, packageFiles)
	err := u.zipRemoveFiles(packageFilename, packageFiles)
	if err != nil {
		return "", err
	}
	return packageFilename, nil
}

func (u *Uploader) UploadPackageFile(packageFile string) error {
	return nil
}

func (u *Uploader) UpdateDirectory(packageFile string, uploadID string, uploadArtist string, uploadTitle string) error {
	return nil
}

func (u *Uploader) copyFile(src, dst string) error {
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

func (u *Uploader) zipRemoveFiles(zipFilename string, files []string) error {
	newZipFile, err := os.Create(zipFilename)
	if err != nil {
			return err
	}
	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()
	for _, file := range files {
		filename := u.tempDir + "/" + file
		log.Printf("[uploader] adding file %s to package file %s", filename, zipFilename)
		if err = addFileToZip(zipWriter, filename); err != nil {
				return err
		}
	}
	newZipFile.Close()
	// zip file created, remove original files
	for _, file := range files {
		filename := u.tempDir + "/" + file
		log.Printf("[uploader] removing file %s", filename)
		err = os.Remove(filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
			return err
	}
	defer fileToZip.Close()
	info, err := fileToZip.Stat()
	if err != nil {
			return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
			return err
	}
	header.Name = filename
	header.Method = zip.Deflate
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
			return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

/*

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
*/