package parser

import "fmt"

type ParserError struct {
	parent   error
	message  string
	location [2]int
}

var _ error = (*ParserError)(nil)

func NewParserError(msg string, location [2]int) ParserError {
	return ParserError{
		parent:   nil,
		message:  msg,
		location: location,
	}
}

func (e ParserError) Error() string {
	return fmt.Sprintf("%s at %d:%d", e.message, e.location[0], e.location[1])
}

func (e ParserError) Location() [2]int { return e.location }
