package mopidy

import (
	"log"
	"os/exec"

	rpc "github.com/ybbus/jsonrpc"
)

const (
	// PlaybackStateStopped indicates playback is stopped.
	PlaybackStateStopped = "stopped"
	// PlaybackStatePlaying indicates playback is playing.
	PlaybackStatePlaying = "playing"
)

// Client is the client for the audio service.
type Client struct {
	rpcClient rpc.RPCClient
}

// Artist represents an artist.
type Artist struct {
	Name string `json:"name"`
}

// Album represents an album.
type Album struct {
	Name string `json:"name"`
	NumTracks int `json:"num_tracks"`
}

// Track represents a single track.
type Track struct {
	Album Album `json:"album"`
	Artists []Artist `json:"artists"`
	Name string `json:"name"`
	URI string `json:"uri"`
}

type responseSearchResult struct {
	Tracks []Track `json:"tracks"`
}

type responseGetTracklist struct {
	Track Track `json:"track"`
}

type payloadQueryAttributes struct {
	Artist string `json:"artist"`
	Album string `json:"album"`
}

type payloadQuery struct {
	Query payloadQueryAttributes `json:"query"`
}

type payloadTracklistAdd struct {
	Uris []string `json:"uris"`
}

// NewClient returns a new client instance.
func NewClient(url string) (*Client, error) {
	client := new(Client)
	client.rpcClient = rpc.NewClient(url)
	return client, nil
}

// AudiobookAvailable queries the given data and returns if the album 
// is known. Returns the number of tracks in the album.
func (c *Client) AudiobookAvailable(artist string, album string) (bool, int, error) {
	resp, err := c.rpcClient.Call("core.library.search", &payloadQuery{Query: payloadQueryAttributes{Artist: artist, Album: album}})
	if err != nil {
		return false, -1, err
	}
	var result *[]responseSearchResult
	err = resp.GetObject(&result)
	if err != nil || result == nil || len(*result) != 1 {
		return false, -1, err
	}
	return true, len((*result)[0].Tracks), nil
}

// AddToTracklist adds the given track URIs to the tracklist.
func (c *Client) AddToTracklist(tracks []string) error {
	log.Printf("[player] adding to tracklist: %s", tracks)
	_, err := c.rpcClient.Call("core.tracklist.add", &payloadTracklistAdd{tracks})
	return err
}

// GetPlaybackState returns the current playback state.
func (c *Client) GetPlaybackState() (string, error) {
	resp, err := c.rpcClient.Call("core.playback.get_state")
	if err != nil {
		return "", err
	}
	result, err := resp.GetString()
	if err != nil {
		return "", err
	}
	return result, nil
}

// GetCurrentTracklist returns the current tracklist.
func (c *Client) GetCurrentTracklist() ([]Track, error) {
	resp, err := c.rpcClient.Call("core.tracklist.get_tl_tracks")
	if err != nil {
		return nil, err
	}
	var result *[]responseGetTracklist
	err = resp.GetObject(&result)
	if err != nil || result == nil {
		return nil, err
	}
	tracks := make([]Track, len(*result))
	for _, entry := range *result {
		tracks = append(tracks, entry.Track)
	}
	return tracks, nil
}

// GetCurrentTrack returns the current track.
func (c *Client) GetCurrentTrack() (*Track, error) {
	resp, err := c.rpcClient.Call("core.playback.get_current_track")
	if err != nil {
		return nil, err
	}
	var result *Track
	err = resp.GetObject(&result)	
	if err != nil || result == nil {
		return nil, err
	}
	return result, nil
}

// RefreshLibrary refreshes the local library.
func (c *Client) RefreshLibrary() error {
	err := exec.Command("sudo", "mopidyctl", "local", "scan").Run()
	if err != nil {
		return err
	}
	_, err = c.rpcClient.Call("core.library.refresh")
	return err
}

// ClearTracklist clears the current tracklist.
func (c *Client) ClearTracklist() error {
	_, err := c.rpcClient.Call("core.tracklist.clear")
	return err
}

// Play plays the current tracklist.
func (c *Client) Play() error {
	_, err := c.rpcClient.Call("core.playback.play")
	return err
}

// Stop stops playback.
func (c *Client) Stop() error {
	_, err := c.rpcClient.Call("core.playback.stop")
	return err
}

