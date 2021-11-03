package marshal

import (
	"bufio"
	"io"
)

// Marshaler is the interface implemented by types that can marshal themselves
// into valid Skyhash.
type Marshaler interface {
	MarshalSkyhash(io.Writer) error
}

// Unmarshaler is the interface implemented by types that can unmarshal a Skyhash
// description of themselves. UnmarshalSkyhash should _always_ fully consume a Skyhash
// message off the reader, unless there is an error returned from the reader
// itself.
//
// Note that, unlike Marshaler, Unmarshaler _must_ take in a *bufio.Reader.
type Unmarshaler interface {
	UnmarshalSkyhash(*bufio.Reader) error
}

// ErrDiscarded is used to wrap an error encountered while unmarshaling a
// message. If an error was encountered during unmarshaling but the rest of the
// message was successfully discarded off of the wire, then the error can be
// wrapped in this type.
type ErrDiscarded struct {
	Err error
}

func (ed ErrDiscarded) Error() string {
	return ed.Err.Error()
}

// Unwrap implements the errors.Wrapper interface.
func (ed ErrDiscarded) Unwrap() error {
	return ed.Err
}
