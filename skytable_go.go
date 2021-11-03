// Package skytable_go implements all functionality needed to work with Skytable
//
// Creating a client
//
// For a single node Skytable instance use NewPool to create a connection pool. The
// connection pool is thread-safe and will automatically create, reuse, and
// recreate connections as needed:
//
//	pool, err := skytable-go.NewPool("tcp", "127.0.0.1:6379", 10)
//	if err != nil {
//		// handle error
//	}
//
// Commands
//
// Any Skytable command can be performed by passing a Cmd into a Client's Do
// method. Each Cmd should only be used once. The return from the Cmd can be
// captured into any appopriate go primitive type, or a slice, map, or struct,
// if the command returns an array.
//
//	err := client.Do(skytable-go.Cmd(nil, "SET", "foo", "someval"))
//
//	var fooVal string
//	err := client.Do(skytable-go.Cmd(&fooVal, "GET", "foo"))
//
//	var fooValB []byte
//	err := client.Do(skytable-go.Cmd(&fooValB, "GET", "foo"))
//
//	var barI int
//	err := client.Do(skytable-go.Cmd(&barI, "INCR", "bar"))
//
//	var bazEls []string
//	err := client.Do(skytable-go.Cmd(&bazEls, "LRANGE", "baz", "0", "-1"))
//
//	var buzMap map[string]string
//	err := client.Do(skytable-go.Cmd(&buzMap, "HGETALL", "buz"))
//
// Errors
//
// Errors returned from Skytable can be explicitly checked for using the the
// resp2.Error type. Note that the errors.As function, introduced in go 1.13,
// should be used.
//
//	var SkytableErr resp2.Error
//	err := client.Do(skytable-go.Cmd(nil, "AUTH", "wrong password"))
//	if errors.As(err, &SkytableErr) {
//		log.Printf("Skytable error returned: %s", SkytableErr.E)
//	}
//
// Use the golang.org/x/xerrors package if you're using an older version of go.
//
package skytable_go

import (
	"errors"
)

var errClientClosed = errors.New("client is closed")

// Client describes an entity which can carry out Actions, e.g. a connection
// pool for a single Skytable instance or the cluster client.
//
// Implementations of Client are expected to be thread-safe, except in cases
// like Conn where they specify otherwise.
type Client interface {
	// Do performs an Action, returning any error.
	Do(Action) error

	// Close is called all future method calls on the Client will return
	// an error
	Close() error
}

// ClientFunc is a function which can be used to create a Client for a single
// Skytable instance on the given network/address.
type ClientFunc func(network, addr string) (Client, error)
