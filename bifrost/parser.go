package bifrost

// File bifrost/parser contains the bifrost.Parser interface.

// Parser is the interface of types containing Bifrost parser and emitter functionality.
// Parsers can convert a set of message structs (here represented as the empty interface) to and from Bifrost messages.
type Parser interface {
	// ParseBifrostRequest parses a Bifrost request with command word and arguments args.
	ParseBifrostRequest(word string, args []string) (interface{}, error)

	// EmitBifrostResponse converts resp into a Bifrost message with the given tag, and sends it through out.
	EmitBifrostResponse(tag string, resp interface{}, out chan<- Message) error
}
