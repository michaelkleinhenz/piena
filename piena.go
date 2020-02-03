package main

import (
	"fmt"
	"log"

	"github.com/fuzxxl/nfc/2.0/nfc"
	"rsc.io/quote"
)

var (
	m      = nfc.Modulation{Type: nfc.ISO14443a, BaudRate: nfc.Nbr106}
	devstr = "" // Use empty string to select first device
)

// Blocks until a target is detected and returns its UID.
// Only cares about the first target it sees.
func get_card(pnd *nfc.Device) ([10]byte, error) {
	for {
		targets, err := pnd.InitiatorListPassiveTargets(m)
		if err != nil {
			return [10]byte{}, fmt.Errorf("listing available nfc targets", err)
		}

		for _, t := range targets {
			if card, ok := t.(*nfc.ISO14443aTarget); ok {
				return card.UID, nil
			}
		}
	}
}

func Hello() string {
	return quote.Hello()
}

func main() {
	fmt.Println("using libnfc", nfc.Version())

	pnd, err := nfc.Open(devstr)
	if err != nil {
		log.Fatalf("could not open device: %v", err)
	}
	defer pnd.Close()

	if err := pnd.InitiatorInit(); err != nil {
		log.Fatalf("could not init initiator: %v", err)
	}

	fmt.Println("opened device", pnd, pnd.Connection())

	card_id, err := get_card(&pnd)
	if err != nil {
		fmt.Errorf("failed to get_card", err)
	}

	if card_id != [10]byte{} {
		fmt.Printf("card found %#X\n", card_id)
	} else {
		fmt.Printf("no card found\n")
	}
}
