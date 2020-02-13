package state

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	// create temp directory
	path, err := ioutil.TempDir("", "piena-")
	assert.NoError(t, err)
	defer os.RemoveAll(path)
	// create serialized store
	state := []AudiobookState{
		AudiobookState{ ID: "111", CurrentOrd: 1 },
		AudiobookState{ ID: "222", CurrentOrd: 2 },
		AudiobookState{ ID: "333", CurrentOrd: 3 },
		AudiobookState{ ID: "444", CurrentOrd: 4 },
		AudiobookState{ ID: "555", CurrentOrd: 5 },
	}
	stateBytes, err := json.MarshalIndent(state, "", " ")
	assert.NoError(t, err)
	err = ioutil.WriteFile(path + "/" + "store.json", stateBytes, 0644)
	assert.NoError(t, err)
	// start testing
	stateStore, err := NewState(path + "/" + "store.json")
	assert.NoError(t, err)
	for _, entry := range state {
		result, err := stateStore.Get(entry.ID)
		assert.NoError(t, err)
		assert.Equal(t, result, entry.CurrentOrd)
	}
	// updated entry
	assert.NoError(t, stateStore.Set("111", 42))
	result, err := stateStore.Get("111")
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	// new entry
	assert.NoError(t, stateStore.Set("999", 23))
	result, err = stateStore.Get("999")
	assert.NoError(t, err)
	assert.Equal(t, 23, result)
	// create another store
	stateStore, err = NewState(path + "/" + "store.json")
	assert.NoError(t, err)
	result, err = stateStore.Get("111")
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	result, err = stateStore.Get("999")
	assert.NoError(t, err)
	assert.Equal(t, 23, result)
	result, err = stateStore.Get("222")
	assert.NoError(t, err)
	assert.Equal(t, 2, result)
}
