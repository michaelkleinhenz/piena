package mopidy

import (
	rpc "github.com/ybbus/jsonrpc"

	"github.com/michaelkleinhenz/piena/base"
)

// Client is the client for the audio service.
type Client struct {
	rpcClient rpc.RPCClient
}

// NewClient returns a new client instance.
func NewClient(url string) (*Client, error) {
	client := new(Client)

	client.rpcClient = rpc.NewClient(url)
	/*
	type Person struct {
    Id   int `json:"id"`
    Name string `json:"name"`
    Age  int `json:"age"`
	}
  var person *Person
	rpcClient.CallFor(&person, "getPersonById", 4711)
	person.Age = 33
	rpcClient.Call("updatePerson", person)
	rpcClient.Call("createPerson", &Person{"Alex", 33, "Germany"})
  // generates body: {"jsonrpc":"2.0","method":"createPerson","params":{"name":"Alex","age":33,"country":"Germany"},"id":0}
	*/

	return client, nil
}

// Close terminates the service connection.
func (c *Client) Close() {
}

// Play starts the album in the given path.
func (c *Client) start(audiobook *base.Audiobook) error {
	return nil
}

// Stop stops playback.
func (c *Client) Stop() error {
	return nil
}

// Continue resumes playback of a prior album.
func (c *Client) Continue(audiobook *base.Audiobook, ord int) error {
	return nil
}


