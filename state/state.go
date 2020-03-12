package state

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
)

// AudiobookState stores the current state of an audiobook
type AudiobookState struct {
	ID         string `json:"id"`
	Artist		 string `json:"artist"`
	Title			 string `json:"title"`
	CurrentOrd int    `json:"currentOrd"`
}

// State manages the current state of an audiobook
type State struct {
	states []AudiobookState
	filepath string
}

// NewState creates a new State instance.
func NewState(filepath string) (*State, error) {
	state := new(State)
	state.filepath = filepath
	state.states = []AudiobookState{}
	if _, err := os.Stat(filepath); err == nil {
		jsonFile, err := os.Open(filepath)
		if err != nil {
			return nil, err
		}
		defer jsonFile.Close()
		byteValue, _ := ioutil.ReadAll(jsonFile)
		json.Unmarshal(byteValue, &state.states)
	}
	return state, nil	
}

// Exists checks if a state exists.
func (s *State) Exists(audiobookID string) bool {
	for idx := range s.states {
		if s.states[idx].ID == audiobookID {
			return true
		}
	}
	return false
}

// SetOrd stores a state.
func (s *State) SetOrd(audiobookID string, ord int) error {
	for idx := range s.states {
		if s.states[idx].ID == audiobookID {
			log.Printf("[state] updating ord %d for audiobook %s", ord, audiobookID)
			s.states[idx].CurrentOrd = ord
			return s.store()
		}
	}
	return errors.New("[store] audiobook not known, store inital record first with Set()")
}

// Set stores a state.
func (s *State) Set(audiobookID string, artist string, title string, ord int) error {
	log.Printf("[state] storing ord %d for audiobook %s", ord, audiobookID)
	for idx := range s.states {
		if s.states[idx].ID == audiobookID {
			s.states[idx].CurrentOrd = ord
			return s.store()
		}
	}
	s.states = append(s.states, AudiobookState{
		ID: audiobookID,
		Artist: artist,
		Title: title,
		CurrentOrd: ord,
	})
	return s.store()
}

// Remove removes a state.
func (s *State) Remove(audiobookID string) error {
	for idx := range s.states {
		if s.states[idx].ID == audiobookID {
			log.Printf("[state] removing state for audiobook %s", audiobookID)
			s.states = append(s.states[:idx], s.states[idx+1:]...)
			return s.store()
		}
	}
	return errors.New("[store] audiobook not found in state store")
}


// Get retrieves a state.
func (s *State) Get(audiobookID string) (int, error) {
	for _, entry := range s.states {
		if entry.ID == audiobookID {
			return entry.CurrentOrd, nil
		}
	}
	return -1, errors.New("[store] audiobook not found in state store")
}

// GetArtistAndTitle retrieves artist and title from the ID.
func (s *State) GetArtistAndTitle(audiobookID string) (string, string, error) {
	for _, entry := range s.states {
		if entry.ID == audiobookID {
			return entry.Artist, entry.Title, nil
		}
	}
	return "", "", errors.New("audiobook not found in state store")
}

func (s *State) store() error {
	stateBytes, err := json.MarshalIndent(s.states, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.filepath, stateBytes, 0644)
}