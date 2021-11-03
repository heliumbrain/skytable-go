package skytable_go

import "github.com/heliumbrain/skytable-go/skyhash"

// Action performs a task using a Conn.
type Action interface {
	// Keys returns the keys which will be acted on. Empty slice or nil may be
	// returned if no keys are being acted on. The returned slice must not be
	// modified.
	Keys() []string

	// Run actually performs the Action using the given Conn.
	Run(c Conn) error
}

type CmdAction interface {
	Action
	skyhash.Marshaler
	skyhash.Unmarshaler
}

func (c *client) Get() {

}

func (c *client) Set() {

}
