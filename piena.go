package main

import (
	"log"

	r "github.com/michaelkleinhenz/piena/reader"
	m "github.com/michaelkleinhenz/piena/mopidy"
	s "github.com/michaelkleinhenz/piena/state"
	d "github.com/michaelkleinhenz/piena/downloader"
)

var (
	nfcReader *r.NfcReader
	channel chan *r.NfcReadResult
	player *m.Client
	state *s.State
	downloader *d.Downloader
)

func main() {
	log.Println("[main] piena starting..")
	var err error

	// initialize nfc reader hardware.
	nfcReader, channel, err = r.NewNfcReader()
	if err != nil {
		log.Fatalf("[main] error initializing nfc hardware: %s\n", err.Error())
	}
	defer nfcReader.Close()

	// initialize mopidy connection.
	player, err = m.NewClient("http://service/rpc")
	err = player.RefreshLibrary()
	if err != nil {
		log.Fatalf("[main] error initializing mopidy connector: %s\n", err.Error())
	}
	err = player.Stop()
	if err != nil {
		log.Fatalf("[main] error initializing mopidy connector: %s\n", err.Error())
	}
	err = player.ClearTracklist()
	if err != nil {
		log.Fatalf("[main] error initializing mopidy connector: %s\n", err.Error())
	}

	// initialize persistence
	state, err = s.NewState("state.json")
	if err != nil {
		log.Fatalf("[main] error initializing persistence state: %s\n", err.Error())
	}

	// initialize downloader
	downloader, err = d.NewDownloader("libPath", "http://directory.url")
	if err != nil {
		log.Fatalf("[main] error initializing downloader: %s\n", err.Error())
	}

	// start processing loop.
	for {
		event := <-channel
		switch event.Result {
		case r.NfcStateError:
			log.Printf("[main] error reading from nfc hardware: %s\n", event.Err.Error())
		case r.NfcStateTagNotPresent:
			log.Println("[main] tag removed")
			err = tagRemoved()
			if err != nil {
				log.Printf("[main] error when removing tag: %s\n", err.Error())
			}
		case r.NfcStateTagPresent:
			log.Printf("[main] tag detected: %s\n", event.ID)
			err = tagDetected(event.ID)
			if err != nil {
				log.Printf("[main] error when processing detected tag: %s\n", err.Error())
			}
		}
	}
}

func tagRemoved() error {
	currentTrack, err := player.GetCurrentTrack()
	if err != nil {
		log.Printf("[main] error getting current track: %s\n", err.Error())
	}
	id, err := downloader.GetID(currentTrack.Album.Artist, currentTrack.Album.Name)
	if err != nil {
		log.Printf("[main] error getting ID for track: %s\n", err.Error())
	}
	if state.Exists(id) {
		err = state.SetOrd(id, currentTrack.Ord)
		if err != nil {
			log.Printf("[main] error storing updated track state: %s\n", err.Error())
		}	
	} else {
		err = state.Set(id, currentTrack.Album.Artist, currentTrack.Album.Name, currentTrack.Ord)
		if err != nil {
			log.Printf("[main] error storing initial track state: %s\n", err.Error())
		}	
	}
	return player.Stop()
}

func tagDetected(ID string) error {
	// retrieve book from ID
	// TODO: display retrieval progress on UX
	audiobook, err := downloader.GetAudiobook(ID)
	if err != nil {
		log.Printf("[main] error retrieving audiobook: %s\n", err.Error())
		return err
	}	
	// if new, store initial dataset in store, else retrieve position
	ord := 1
	if !state.Exists(ID) {
		state.Set(ID, audiobook.Artist, audiobook.Title, 1)
	} else {
		ord, err := state.Get(ID)
		if err != nil {
			log.Printf("[main] error retrieving audiobook state: %s\n", err.Error())
			// fallback: start over from track 1
			state.Set(ID, audiobook.Artist, audiobook.Title, ord)
		}		
	}
	// stop current playback and clear tracklist
	player.Stop()
	player.ClearTracklist()
	// add new tracks to tracklist from the retrieved ord
	// TODO: make sure tracks are now in the library, refresh if needed
	err = player.RefreshLibrary()
	if err != nil {
		log.Printf("[main] error refreshing track library: %s\n", err.Error())
		return err
	}
	tracklist := string[]{}
	for idx, track := range(audiobook.Tracks) {
		if idx >= ord {
			tracklist = append(tracklist, track.Filename)
		}
	}
	err = player.AddToTracklist(tracklist)
	if err != nil {
		log.Printf("[main] error adding tracks to tracklist: %s\n", err.Error())
		return err
	}
	// start playback
	err = player.Play()
	if err != nil {
		log.Printf("[main] error starting playback: %s\n", err.Error())
		return err
	}
	return nil
}