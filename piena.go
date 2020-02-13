package main

import (
	"log"

	"github.com/michaelkleinhenz/piena/reader"
	"github.com/michaelkleinhenz/piena/mopidy"
)

func main() {
	log.Println("[main] piena starting..")
	// initialize nfc reader hardware.
	nfcReader, channel, err := reader.NewNfcReader()
	if err != nil {
		log.Fatalf("[main] error initializing nfc hardware: %s\n", err.Error())
	}
	defer nfcReader.Close()
	// initialize mopidy connection.
	player, err := NewClient("http://service/rpc")
	defer player.Close()
	player.Stop()

	// start processing loop.
	for {
		event := <-channel
		switch event.Result {
		case reader.NfcStateError:
			log.Printf("[main] error reading from nfc hardware: %s\n", event.Err.Error())
		case reader.NfcStateTagNotPresent:
			log.Println("[main] tag removed")
		case reader.NfcStateTagPresent:
			log.Printf("[main] tag detected: %s\n", event.ID)
		}
	}
}
