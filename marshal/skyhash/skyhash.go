package skyhash

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/heliumbrain/skytable-go/internal/bytesutil"
	"github.com/heliumbrain/skytable-go/marshal"
)

var delim = []byte{'\n'}

// prefix enumerates the possible Skyhash types by enumerating the different
// prefixes a Skyhash message might start with.
type prefix []byte

// Enumeration of each of Skyhash message types, each denoted by the prefix which
// is prepended to messages of that type.
//
// In order to determine the type of a message which is being written to a
// *bufio.Reader, without actually consuming it, one can use the Peek method and
// compare it against these values.
var (
	StringPrefix         = []byte{'+'}
	SmallIntPrefix       = []byte{'.'}
	SmallIntSignedPrefix = []byte{'-'}
	IntPrefix            = []byte{':'}
	IntSignedPrefix      = []byte{';'}
	FloatPrefix          = []byte{'%'}
	JSONPrefix           = []byte{'$'}
	BlobPrefix           = []byte{'?'}
	ResponsePrefix       = []byte{'!'}
	AnyArrayPrefix       = []byte{'~'}
	ErrorPrefix          = []byte{'1'} // This needs to be specified further for different errors
)

// String formats a prefix into a human-readable name for the type it denotes.
func (p prefix) String() string {
	pStr := string(p)
	switch pStr {
	case string(StringPrefix):
		return "string"
	case string(SmallIntPrefix):
		return "small-integer"
	case string(SmallIntSignedPrefix):
		return "small-integer-signed"
	case string(IntPrefix):
		return "integer"
	case string(IntSignedPrefix):
		return "integer-signed"
	case string(FloatPrefix):
		return "float"
	case string(JSONPrefix):
		return "json"
	case string(BlobPrefix):
		return "blob"
	case string(ResponsePrefix):
		return "response"
	case string(AnyArrayPrefix):
		return "any-array"
	case string(ErrorPrefix):
		return "error"
	default:
		return pStr
	}
}

var (
	simpleQuery = []byte("*1\n")
)

var bools = [][]byte{
	{'0'},
	{'1'},
}

////////////////////////////////////////////////////////////////////////////////

type errUnexpectedPrefix struct {
	Prefix         []byte
	ExpectedPrefix []byte
}

func (e errUnexpectedPrefix) Error() string {
	return fmt.Sprintf(
		"expected prefix %q, got %q",
		prefix(e.ExpectedPrefix).String(),
		prefix(e.Prefix).String(),
	)
}

// peekAndAssertPrefix will peek at the next incoming redis message and assert
// that it is of the type identified by the given Skyhash prefix
// If the prefix is not the expected one, an error is returned.
func peekAndAssertPrefix(br *bufio.Reader, expectedPrefix []byte) error {
	b, err := br.Peek(len(expectedPrefix))
	if err != nil {
		return err
	} else if bytes.Equal(b, expectedPrefix) {
		return nil
	} else if bytes.Equal(b, ErrorPrefix) {
		var respErr Error
		if err := respErr.UnmarshalSkyhash(br); err != nil {
			return err
		}
		return marshal.ErrDiscarded{Err: respErr}
	}

	return marshal.ErrDiscarded{Err: errUnexpectedPrefix{
		Prefix:         b,
		ExpectedPrefix: expectedPrefix,
	}}
}

// like peekAndAssertPrefix, but will consume the prefix if it is the correct
// one as well.
func assertBufferedPrefix(br *bufio.Reader, pref []byte) error {
	if err := peekAndAssertPrefix(br, pref); err != nil {
		return err
	}
	_, err := br.Discard(len(pref))
	return err
}

////////////////////////////////////////////////////////////////////////////////

//SkytableString Represents the Skytable string type
type SkytableString struct {
	S string
}

func (ss SkytableString) MarshalSkyhash(w io.Writer) error {
	stringlength := len(ss.S)

	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, StringPrefix...)
	*scratch = append(*scratch, []byte(string(rune(stringlength)))...)
	*scratch = append(*scratch, delim...)
	*scratch = append(*scratch, ss.S...)
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

func (ss SkytableString) UnmarshalSkyhash(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, StringPrefix); err != nil {
		return err
	}
	b, err := bytesutil.BufferedBytesDelim(br)
	if err != nil {
		return err
	}

	ss.S = string(b)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

type AnyArray struct {
	A []string
}

////////////////////////////////////////////////////////////////////////////////

// Error represents an error type in the Skyhash protocol. Note that this only
// represents an actual error message being read/written on the stream, it is
// separate from network or parsing errors. An E value of nil is equivalent to
// an empty error string.
type Error struct {
	E error
}

func (e Error) Error() string {
	return e.E.Error()
}

// MarshalSkyhash implements the Marshaler method.
func (e Error) MarshalSkyhash(w io.Writer) error {
	scratch := bytesutil.GetBytes()
	*scratch = append(*scratch, ErrorPrefix...)
	if e.E != nil {
		*scratch = append(*scratch, e.E.Error()...)
	}
	*scratch = append(*scratch, delim...)
	_, err := w.Write(*scratch)
	bytesutil.PutBytes(scratch)
	return err
}

// UnmarshalSkyhash implements the Unmarshaler method.
func (e *Error) UnmarshalSkyhash(br *bufio.Reader) error {
	if err := assertBufferedPrefix(br, ErrorPrefix); err != nil {
		return err
	}
	b, err := bytesutil.BufferedBytesDelim(br)
	e.E = errors.New(string(b))
	return err
}

// As implements the method for the (x)errors.As function.
func (e Error) As(target interface{}) bool {
	switch targetT := target.(type) {
	case *marshal.ErrDiscarded:
		targetT.Err = e
		return true
	default:
		return false
	}
}

////////////////////////////////////////////////////////////////////////////////////////////
