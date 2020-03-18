package main

import (
	"flag"
	"log"
	"strings"
	"time"

	d "github.com/michaelkleinhenz/piena/downloader"
	m "github.com/michaelkleinhenz/piena/mopidy"
	r "github.com/michaelkleinhenz/piena/reader"
	s "github.com/michaelkleinhenz/piena/state"
  u "github.com/michaelkleinhenz/piena/uploader"
)

var (
	nfcReader *r.NfcReader
	channel chan *r.NfcReadResult
	player *m.Client
	state *s.State
	downloader *d.Downloader
	// TODO: this should be the complete audiobook
	lastSeenID string = ""
)

func main() {
	playerPtr := flag.String("playerurl", "http://localhost:6680/mopidy/rpc", "Mopidy RPC endpoint address")
	libraryURLPtr := flag.String("libraryurl", "http://d3aj4nh2mw9ghj.cloudfront.net/directory.json", "Audiobook library URL")
	libraryDirectoryPtr := flag.String("librarypath", "/home/pi/audiobooks", "Audiobook local library path")
	uploadPtr := flag.Bool("upload", false, "Upload audiobook to backend service")
	uploadFileDir := flag.String("dir", "", "Directory with files to be uploaded")
	uploadArtist := flag.String("artist", "", "Artist for uploaded files")
	uploadTitle := flag.String("title", "", "Title for uploaded files")
	uploadID := flag.String("id", "", "ID for uploaded files")
	uploadBucket := 	flag.String("s3bucked", "tiena-files", "S3 bucket for upload")
	flag.Parse()
	
	log.Println("[main] piena starting..")
  var err error

	// check if we should upload an audiobook.
	if *uploadPtr {
		if *uploadArtist == "" || *uploadFileDir == "" || *uploadID == "" || *uploadTitle == "" {
			log.Fatal("[main] Not all required parameters given for file upload")
		}
		uploader, err := u.NewUploader()
		packageFiles, err := uploader.TagRenameFiles(*uploadFileDir, *uploadArtist, *uploadTitle)
		if err != nil {
			log.Fatalf("[main] error tagging upload files: %s", err.Error())
		}
		packageFile, err := uploader.PackageFiles(packageFiles, *uploadArtist, *uploadTitle)
		if err != nil {
			log.Fatalf("[main] error packaging upload files: %s", err.Error())
		}
		err = uploader.UploadPackageFile(packageFile, *uploadBucket)
		if err != nil {
			log.Fatalf("[main] error uploading package file: %s", err.Error())
		}
		err = uploader.UpdateDirectory(packageFile, *uploadID, *uploadArtist, *uploadTitle, *uploadBucket)
		if err != nil {
			log.Fatalf("[main] error updating directory: %s", err.Error())
		}
		log.Println("[main] upload completed")
		return
	}

	// initialize nfc reader hardware.
	nfcReader, channel, err = r.NewNfcReader()
  for err != nil {
		log.Printf("[main] error initializing nfc hardware: %s, retrying..", err.Error())
		nfcReader, channel, err = r.NewNfcReader()
	}
	defer nfcReader.Close()

	// initialize mopidy connection.
	player, err = m.NewClient(*playerPtr)
	err = player.RefreshLibrary()
	if err != nil {
		log.Fatalf("[main] error initializing mopidy connector: %s", err.Error())
	}
	err = player.Stop()
	if err != nil {
		log.Fatalf("[main] error initializing mopidy connector: %s", err.Error())
	}
	err = player.ClearTracklist()
	if err != nil {
		log.Fatalf("[main] error initializing mopidy connector: %s", err.Error())
	}

	// initialize persistence
	state, err = s.NewState("state.json")
	if err != nil {
		log.Fatalf("[main] error initializing persistence state: %s", err.Error())
	}

	// initialize downloader
	downloader, err = d.NewDownloader(*libraryDirectoryPtr, *libraryURLPtr)
	if err != nil {
		log.Fatalf("[main] error initializing downloader: %s", err.Error())
	}

	// start gofunc that polls current track and updates state
	go func() {
		for {
			currentTrack, err := player.GetCurrentTrack()
			if err != nil {
				log.Fatalf("[main] error getting current track in polling loop: %s", err.Error())
			}
			if currentTrack != nil {
				log.Printf("[main] polling loop: current track is %s", currentTrack.URI)
				id, ord, err := getIdAndOrdForCurrentTrack(currentTrack)
				if err != nil {
					log.Printf("[main] error or unknown track when getting current id and ord in polling loop: %s", err.Error())
				} else {
					log.Printf("[main] storing updated ord %d for audiobook %s in polling loop", ord, id)
					lastSeenID = id
					if state.Exists(id) {
						err = state.SetOrd(id, ord)
						if err != nil {
							log.Printf("[main] error storing updated track state in polling loop: %s", err.Error())
						}	
					} else {
						err = state.Set(id, currentTrack.Artists[0].Name, currentTrack.Album.Name, ord)
						if err != nil {
							log.Printf("[main] error storing initial track state in polling loop: %s", err.Error())
						}	
					}		
				}
			} else {
				// we are likely at the end of the playlist, remove state for lastSeenID
				if lastSeenID != "" {
					log.Printf("[main] polling loop: removing state for audiobook %s", lastSeenID)
					err = state.Remove(lastSeenID)
					if err != nil {
						log.Printf("[main] error removing state in polling loop: %s", err.Error())
					}	
					// we also remove the tracklist in this case
				  err = player.ClearTracklist()
					if err != nil {
						log.Printf("[main] error clearing tracklist in polling loop: %s", err.Error())
					}	
					lastSeenID = ""	
				}
			}
			time.Sleep(1*time.Second)	
		}
	}()

	// start processing loop.
	for {
		event := <-channel
		switch event.Result {
		case r.NfcStateError:
			log.Printf("[main] error reading from nfc hardware: %s", event.Err.Error())
		case r.NfcStateTagNotPresent:
			log.Println("[main] tag removed")
			err = tagRemoved()
			if err != nil {
				log.Printf("[main] error when removing tag: %s", err.Error())
			}
		case r.NfcStateTagPresent:
			log.Printf("[main] tag detected: %s", event.ID)
			err = tagDetected(event.ID)
			if err != nil {
				log.Printf("[main] error when processing detected tag %s: %s", event.ID, err.Error())
			}
		}
	}
}

func getIdAndOrdForCurrentTrack(currentTrack *m.Track) (string, int, error) {
	id, err := downloader.GetID(currentTrack.Artists[0].Name, currentTrack.Album.Name)
	if err != nil {
		return "", -1, err
	}
	audiobook, _, err := downloader.GetAudiobook(id)
	if err != nil {
		return "", -1, err
	}
	ord := 1
	for idx, trackName := range(audiobook.Tracks) {
		if trackName.Title == currentTrack.Name {
			ord = idx+1
		}
	}
	return id, ord, nil
}

func tagRemoved() error {
	currentTrack, err := player.GetCurrentTrack()
	if err != nil {
		log.Printf("[main] error getting current track: %s", err.Error())
	}
	// stop current playback and clear tracklist
	log.Println("[main] stopping and clearing current playlist")
	// TODO: error handling
	player.Stop()
	player.ClearTracklist()
	if currentTrack == nil {
		// no current track, just return
		return nil
	} 
	// get ord from track name (ord is not returned from player)	
	id, ord, err := getIdAndOrdForCurrentTrack(currentTrack)
	if err != nil {
		log.Printf("[main] error getting audiobook for id: %s", err.Error())
	}
	log.Printf("[main] current track of audiobook is %d", ord)
	if state.Exists(id) {
		err = state.SetOrd(id, ord)
		if err != nil {
			log.Printf("[main] error storing updated track state: %s", err.Error())
		}	
	} else {
		err = state.Set(id, currentTrack.Artists[0].Name, currentTrack.Album.Name, ord)
		if err != nil {
			log.Printf("[main] error storing initial track state: %s", err.Error())
		}	
	}
	// reset current id
	lastSeenID = ""
	return player.Stop()
}

func tagDetected(ID string) error {
	log.Printf("[main] tag detected: %s", ID)
	// retrieve book from ID
	// TODO: display retrieval progress on UX
	audiobook, alreadyExisted, err := downloader.GetAudiobook(ID)
	if err != nil {
		log.Printf("[main] error retrieving audiobook: %s", err.Error())
		return err
	}	
	// store current id
	lastSeenID = audiobook.ID
	// if new, store initial dataset in store, else retrieve position
	log.Printf("[main] found matching audiobook for id %s: %s %s", ID, audiobook.Artist, audiobook.Title)
	ord := 1
	if !state.Exists(ID) {
		log.Printf("[main] no state exists for audiobook %s", ID)
		state.Set(ID, audiobook.Artist, audiobook.Title, 1)
	} else {
		ord, err = state.Get(ID)
		if err != nil {
			log.Printf("[main] error retrieving audiobook state: %s", err.Error())
			// fallback: start over from track 1
			state.Set(ID, audiobook.Artist, audiobook.Title, ord)
		}		
		log.Printf("[main] state exists for audiobook %s: current track is %d", ID, ord)
	}
	// stop current playback and clear tracklist
	log.Println("[main] stopping and clearing current playlist")
	err = player.Stop()
	if err != nil {
		log.Printf("[main] error stopping playback: %s", err.Error())
		return err
	}	
	err = player.ClearTracklist()
	if err != nil {
		log.Printf("[main] error clearing tracklist: %s", err.Error())
		return err
	}	
	// refresh library if needed
	if alreadyExisted {
		log.Println("[main] audiobook already existed in library, no refreshing necessary")
	} else {
		log.Println("[main] refreshing library")
		err = player.RefreshLibrary()
		if err != nil {
			log.Printf("[main] error refreshing track library: %s", err.Error())
			return err
		}		
	}
	// add new tracks to tracklist from the retrieved ord
	log.Printf("[main] building new tracklist for audiobook %s from ord %d", ID, ord)
	tracklist := []string{}
	for idx, track := range(audiobook.Tracks) {
		if idx >= ord-1 {
			tracklist = append(tracklist, strings.ReplaceAll("local:track:" + audiobook.Artist + "/" + audiobook.Title + "/" + track.Filename, " ", "%20"))
		}
	}
	log.Printf("[main] adding tracklist for audiobook %s to playlist: %s", ID, tracklist)
	err = player.AddToTracklist(tracklist)
	if err != nil {
		log.Printf("[main] error adding tracks to tracklist: %s", err.Error())
		return err
	}
	// start playback
	log.Printf("[main] starting playback for audiobook %s", ID)
	err = player.Play()
	if err != nil {
		log.Printf("[main] error starting playback: %s", err.Error())
		return err
	}
	return nil
}