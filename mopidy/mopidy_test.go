package mopidy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMopidy(t *testing.T) {

	client, err := NewClient("http://streamdeck:6680/mopidy/rpc")
	assert.NoError(t, err)

	/*
	err = client.RefreshLibrary()
	assert.NoError(t, err)

	available, numTracks, err := client.AudiobookAvailable("Jan Tenner", "01 - Ein neuer Anfang")
	assert.NoError(t, err)
	assert.True(t, available)
	assert.Equal(t, 11, numTracks)

	err = client.ClearTracklist()
	assert.NoError(t, err)

	err = client.AddToTracklist([]string{"local:track:01%20-%20Ein%20Neuer%20Anfang/01.mp3"})
	assert.NoError(t, err)

	err = client.Play()
	assert.NoError(t, err)
	*/

	err = client.Stop()
	assert.NoError(t, err)

	/*
		tracks, err := client.GetCurrentTracklist()
		assert.NoError(t, err)
		fmt.Println(tracks)

		track, err := client.GetCurrentTrack()
		assert.NoError(t, err)
		fmt.Println(track)

		client.Stop()
	*/
}
