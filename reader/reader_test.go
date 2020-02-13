package reader

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNfcReader(t *testing.T) {
	TestMode = true
	TestModeResult = NfcStateTagNotPresent
	TestModeID = ""
	reader, channel, err := NewNfcReader()

	// basic saneness.
	require.NoError(t, err)
	require.NotNil(t, channel)
	require.NotNil(t, reader)

    // add tag to reader.
	TestModeResult = NfcStateTagPresent
    TestModeID = "12345678"
    result := <-channel
    require.Equal(t, TestModeID, result.ID)

    // update tag.
	TestModeResult = NfcStateTagPresent
    TestModeID = "ABCDEF"
    result = <-channel
    require.Equal(t, TestModeID, result.ID)

    // remove tag.
    TestModeResult = NfcStateTagNotPresent
    TestModeID = ""
    result = <-channel
    require.Equal(t, NfcStateTagNotPresent, result.Result)

    // terminate reader instance
    reader.Close()
}
