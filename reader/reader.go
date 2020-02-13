package reader

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/michaelkleinhenz/piena/nfc"
)

const (
	// NfcStateError signals an error.
	NfcStateError = -1
	// NfcStateTagNotPresent signals a tag not being present on the reader.
	NfcStateTagNotPresent = 0
	// NfcStateTagPresent signals a tag being present on the reader.
	NfcStateTagPresent = 1
)

var (
	// TestMode signals if the modules is in testing mode.
	TestMode bool = false
	// TestModeResult is the returned result state in testing mode.
	TestModeResult int
	// TestModeID is the returned id in testing mode.
	TestModeID string
)

// NfcReadResult is the result data structure.
type NfcReadResult struct {
	Result int
	ID     string
	Err    error
}

// NfcReader is the hardware reader interface.
type NfcReader struct {
	terminateReader      bool
	currentNfcReadResult *NfcReadResult
	channel              chan *NfcReadResult
}

// NewNfcReader eturns a new nfcReader instance.
func NewNfcReader() (*NfcReader, chan *NfcReadResult, error) {
	log.Printf("[reader] using libnfc version %s\n", nfc.Version())
	r := new(NfcReader)
	r.terminateReader = false
	pnd, err := r.initNfcHardware()
	if err != nil {
		return nil, nil, err
	}
	r.channel = make(chan *NfcReadResult)
	go r.runLoop(pnd, r.channel)
	return r, r.channel, nil
}

// Close terminates the reader hardware.
func (r *NfcReader) Close() {
	log.Println("[reader] terminating reader instance")
	r.terminateReader = true
}

func (r *NfcReader) initNfcHardware() (*nfc.Device, error) {
	if TestMode {
		log.Println("[reader] nfc module in test mode, returning dummy device")
		return &nfc.Device{}, nil
	}
	pnd, err := nfc.Open("")
	if err != nil {
		return nil, err
	}
	if err := pnd.InitiatorInit(); err != nil {
		return nil, err
	}
	log.Printf("[reader] opened nfc reader device %s\n", pnd.Connection())
	pnd.SetPropertyBool(nfc.InfiniteSelect, false)
	return &pnd, nil
}

func (r *NfcReader) toString(t nfc.Target) (string, error) {
	if card, ok := t.(*nfc.ISO14443aTarget); ok {
		return fmt.Sprintf("%#x", card.UID), nil
	}
	return "", errors.New("error converting target to string")
}

func (r *NfcReader) getCurrentNFCTagID(pnd *nfc.Device) *NfcReadResult {
	if TestMode {
		log.Printf("[reader] getCurrentNFCTagID returning dummy result state=%d id=%s\n", TestModeResult, TestModeID)
		time.Sleep(500 * time.Millisecond)
		return &NfcReadResult{
			Result: TestModeResult,
			ID:     TestModeID,
			Err:    nil,
		}
	}
	target, err := pnd.InitiatorSelectPassiveTarget(nfc.Modulation{Type: nfc.ISO14443a, BaudRate: nfc.Nbr106}, nil)
	if err != nil {
		return &NfcReadResult{Result: NfcStateError, ID: "", Err: err}
	}
	if target == nil {
		return &NfcReadResult{Result: NfcStateTagNotPresent, ID: "", Err: err}
	}
	tagID, err := r.toString(target)
	if err != nil {
		return &NfcReadResult{Result: NfcStateError, ID: "", Err: err}
	}
	return &NfcReadResult{Result: NfcStateTagPresent, ID: tagID, Err: nil}
}

func (r *NfcReader) runLoop(pnd *nfc.Device, c chan *NfcReadResult) {
	// when this terminates, we also close the channel and device.
	defer func() {
		close(c)
		if !TestMode {
			pnd.Close()
		}
	}()
	// as long as terminateReader is false, we run in a loop.
	for !r.terminateReader {
		readResult := r.getCurrentNFCTagID(pnd)
		if readResult.Err != nil {
			// read returned an error, remove current result, return error.
			log.Printf("[reader] error reading from nfc reader: %s\n", readResult.Err.Error())
			r.currentNfcReadResult = nil
			c <- readResult
		} else {
			// read returned no error, check status.
			switch readResult.Result {
			case NfcStateTagPresent:
				if !r.safeCompareReadResults(r.currentNfcReadResult, readResult) {
					log.Printf("[reader] detected updated ID on reader (old=%s, new=%s)\n", r.serialize(r.currentNfcReadResult), r.serialize(readResult))
					r.currentNfcReadResult = readResult
					c <- readResult
				}
			case NfcStateTagNotPresent:
				if r.currentNfcReadResult != nil {
					log.Printf("[reader] detected removed tag on reader (old=%s)\n", r.serialize(r.currentNfcReadResult))
					r.currentNfcReadResult = nil
					c <- readResult
				}
			}
		}
	}
	log.Println("[reader] terminated reader instance")
}

func (r *NfcReader) serialize(a *NfcReadResult) string {
	if a == nil {
		return "nil"
	}
	return strconv.Itoa(a.Result) + ":" + a.ID
}

func (r *NfcReader) safeCompareReadResults(a *NfcReadResult, b *NfcReadResult) bool {
	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a.Result == b.Result && a.ID == b.ID {
		return true
	}
	return false
}
