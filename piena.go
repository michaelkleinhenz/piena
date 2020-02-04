package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/michaelkleinhenz/piena/nfc"
)

const (
	NFC_STATE_ERROR = -1
	NFC_STATE_TAGNOTPRESENT = 0
	NFC_STATE_TAGPRESENT = 1
)

var (
	nfcModulationType = nfc.Modulation{Type: nfc.ISO14443a, BaudRate: nfc.Nbr106}
	currentNFCTarget  nfc.Target
)

func toString(t nfc.Target) (string, error) {
	if card, ok := t.(*nfc.ISO14443aTarget); ok {
		return fmt.Sprintf("%#x", card.UID), nil
	} 
	return "", errors.New("error converting target to string")
}

func getCurrentNFCTagID(pnd *nfc.Device) (int, string, error) {
	target, err := pnd.InitiatorSelectPassiveTarget(nfcModulationType, nil)
	if err != nil {
		return NFC_STATE_ERROR, "", err
	}
	if target == nil {
		return NFC_STATE_TAGNOTPRESENT, "", nil
	}
	tagID, err := toString(target)
	if err != nil {
		return NFC_STATE_ERROR, "", err
	}
	return NFC_STATE_TAGPRESENT, tagID, nil
}

func main() {
	fmt.Println("using libnfc", nfc.Version())

	// open the first available NFC reader.
	pnd, err := nfc.Open("")
	if err != nil {
		log.Fatalf("could not open device: %v", err)
	}
	defer pnd.Close()

	if err := pnd.InitiatorInit(); err != nil {
		log.Fatalf("could not init initiator: %v", err)
	}

	fmt.Println("opened device", pnd, pnd.Connection())
	pnd.SetPropertyBool(nfc.InfiniteSelect, false)
	
	//err = pnd.InitiatorDeselectTarget()
	if err != nil {
		fmt.Errorf("error deselecting tag", err)
	}

	for {
		//resultCode, tagID, err := getCurrentNFCTagID(&pnd)
		resultCode, tagID, err := getCurrentNFCTagID(&pnd)
		if err != nil {
			fmt.Errorf("failed to query reader", err)
		}
		switch resultCode {
		case NFC_STATE_ERROR:
			fmt.Println("Resultcode: error")
		case NFC_STATE_NEWTAGPRESENT:
			fmt.Printf("Resultcode: new tag present: %s\n", tagID)
		case NFC_STATE_NOTAGPRESENT:
			fmt.Println("Resultcode: no tag present")
		case NFC_STATE_TAGREMOVED:
			fmt.Println("Resultcode: tag removed")
		case NFC_STATE_TAGSTILLPRESENT:
			fmt.Printf("Resultcode: tag still present: %s\n", tagID)

		}
	}
}
