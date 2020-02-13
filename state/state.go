package state

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

// AudiobookState stores the current state of an audiobook
type AudiobookState struct {
	ID         string `json:"id"`
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

// Set stores a state.
func (s *State) Set(audiobookID string, ord int) error {
	for idx := range s.states {
		if s.states[idx].ID == audiobookID {
			s.states[idx].CurrentOrd = ord
			return s.store()
		}
	}
	s.states = append(s.states, AudiobookState{
		ID: audiobookID,
		CurrentOrd: ord,
	})
	return s.store()
}

// Get retrieves a state.
func (s *State) Get(audiobookID string) (int, error) {
	for _, entry := range s.states {
		if entry.ID == audiobookID {
			return entry.CurrentOrd, nil
		}
	}
	return -1, errors.New("audiobook not found in state store")
}

func (s *State) store() error {
	stateBytes, err := json.MarshalIndent(s.states, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.filepath, stateBytes, 0644)
}