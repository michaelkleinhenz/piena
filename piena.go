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
	state.Set(TODO_ID, currentTrack.Ord)
	return player.Stop()
}

func tagDetected(ID string) error {
	// TODO:retrieve book from ID
	player.Stop()
	player.ClearTracklist()
	// TODO:put tracks into tracklist
	// TODO:retrieve stored position
	// TODO:start playback from position
	return nil
}