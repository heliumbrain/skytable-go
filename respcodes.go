package skytable_go

type ResponseCode int

const (
	Okay            ResponseCode = iota // Okay = 0
	NotFound                            // NotFound = 1
	OverWriteError                      // OverWriteError = 2
	ActionError                         // ActionError = 3
	PacketError                         // PacketError = 4
	ServerError                         // ServerError = 5
	ErrorString                         // ErrorString = 6
	WrongType                           // WrongType = 7
	UnknownDataType                     // UnknownDataType = 8
	EncodingError                       // EncodingError = 9
)

func (rc ResponseCode) ToString() string {
	return []string{"Okay", "Not Found", "Overwrite Error", "Action Error", "Packet Error", "Server Error", "Error String", "Wrong Type Error", "Unknown Data Type Error", "Encoding Error"}[rc]
}

func (rc ResponseCode) ToResponse() string {
	return "" // TODO!
}
