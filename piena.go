package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/fuzxxl/nfc/2.0/nfc"
)

const (
	NFC_STATE_ERROR           = -1
	NFC_STATE_TAGREMOVED      = 0
	NFC_STATE_TAGSTILLPRESENT = 1
	NFC_STATE_NEWTAGPRESENT   = 2
	NFC_STATE_NOTAGPRESENT    = 3
)

var (
	nfcModulationType = nfc.Modulation{Type: nfc.ISO14443a, BaudRate: nfc.Nbr106}
	currentNFCTarget  nfc.Target
)

func toString(t nfc.Target) (string, error) {
	if card, ok := t.(*nfc.ISO14443aTarget); ok {
		return fmt.Sprintf("%#X", card.UID), nil
	} 
	return "", errors.New("error converting target to string")
}

func getCurrentNFCTagID(pnd *nfc.Device) (int, string, error) {
	targets, err := pnd.InitiatorListPassiveTargets(nfcModulationType)
	if err != nil {
		return NFC_STATE_ERROR, "", err
	}
	if len(targets) == 0 { // no tag or tag still present.
		// check if old tag is still present
		if currentNFCTarget != nil {
			// select the current tag.
			fmt.Println("1")
			_, err := pnd.InitiatorSelectPassiveTarget(nfcModulationType, nil)
			fmt.Println("2")
			if err != nil {
				return NFC_STATE_ERROR, "", err
			}
			fmt.Println("3")
			result := pnd.InitiatorTargetIsPresent(currentNFCTarget)
			fmt.Println("4")
			if result == nil { // success, old tag still present.
				tagID, err := toString(currentNFCTarget)
				if err != nil {
					return NFC_STATE_ERROR, "", err
				}
				return NFC_STATE_TAGSTILLPRESENT, tagID, nil
			} // fail, old tag not present anymore.
			currentNFCTarget = nil
			err = pnd.InitiatorDeselectTarget()
			if err != nil {
				fmt.Errorf("error deselecting tag", err)
			}
			return NFC_STATE_TAGREMOVED, "", result
		}
		// no tag present and no old tag.
		return NFC_STATE_NOTAGPRESENT, "", nil
	} else if len(targets) == 1 { // one new tag detected.
		currentNFCTarget = targets[0]
		fmt.Println(currentNFCTarget)
		uID := (currentNFCTarget.(*nfc.ISO14443aTarget)).UID
		fmt.Println(uID)
		fmt.Println("a1")
		_, err := pnd.InitiatorSelectPassiveTarget(nfcModulationType, uID[0:4])
		fmt.Println("a2")
		tagID, err := toString(currentNFCTarget)
		if err != nil {
			return NFC_STATE_ERROR, "", err
		}
		return NFC_STATE_NEWTAGPRESENT, tagID, nil
	}
	// multiple tags detected.
	return NFC_STATE_ERROR, "", err
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
	err = pnd.InitiatorDeselectTarget()
	if err != nil {
		fmt.Errorf("error deselecting tag", err)
	}

	for {
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
			fmt.Println("Response: ", err)
		case NFC_STATE_TAGSTILLPRESENT:
			fmt.Printf("Resultcode: tag still present: %s\n", tagID)

		}
	}
}
