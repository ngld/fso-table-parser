package parser

import (
	"fmt"
	"io"
	"strings"

	"github.com/rotisserie/eris"
)

//go:generate stringer -type TokenType scanner.go
type TokenType uint16

const (
	HashLabel TokenType = iota + 1
	DollarLabel
	PlusLabel
	Line
	String
	Number
	Comment
	BlockComment
	HashEnd
)

type Token struct {
	Content  string
	Location [2]int
	Type     TokenType
}

func (t Token) Errorf(msg string, args ...interface{}) error {
	return eris.Wrap(NewParserError(fmt.Sprintf(msg, args...), t.Location), "")
}

type Lexer struct {
	buffer io.RuneScanner
	next   *Token
	line   int
	col    int
}

func NewLexer(buffer io.RuneScanner) *Lexer {
	return &Lexer{buffer: buffer}
}

func (l *Lexer) errorf(msg string, args ...interface{}) error {
	return eris.Wrap(NewParserError(fmt.Sprintf(msg, args...), [2]int{l.line + 1, l.col}), "")
}

func (l *Lexer) Peek() (Token, error) {
	if l.next == nil {
		err := l.readToken()
		if err != nil {
			return Token{}, err
		}
	}

	for l.next.Type == Comment {
		err := l.readToken()
		if err != nil {
			return Token{}, err
		}
	}

	return *l.next, nil
}

func (l *Lexer) Next() (Token, error) {
	if l.next == nil {
		err := l.readToken()
		if err != nil {
			return Token{}, err
		}
	}

	for l.next.Type == Comment {
		err := l.readToken()
		if err != nil {
			return Token{}, err
		}
	}

	result := l.next
	l.next = nil
	return *result, nil
}

func (l *Lexer) Consume() error {
	if l.next == nil {
		_, err := l.Next()
		return err
	}

	/*fmt.Print("Consuming: ")
	spew.Dump(l.next)*/
	l.next = nil
	return nil
}

func (l *Lexer) ReadMultilineText(end string) (string, error) {
	result := ""
	if l.next != nil {
		result += l.next.Content
		l.next = nil
	} else {
		char, _, err := l.buffer.ReadRune()
		if err != nil {
			return "", err
		}
		if char == ':' {
			l.col++
			if err = l.skipWhitespace(); err != nil {
				return "", err
			}
		} else {
			if err = l.buffer.UnreadRune(); err != nil {
				return "", err
			}
		}
	}

	endpos := -1
	for {
		char, size, err := l.buffer.ReadRune()
		if err != nil {
			return "", err
		}

		if char == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}

		if endpos > -1 {
			if string(char) == end[endpos:endpos+size] {
				endpos += size
				if endpos >= len(end) {
					return result, nil
				}
			} else {
				result += end[:endpos]
				endpos = -1
			}
		} else {
			if char == '$' {
				endpos = 1
			} else {
				result += string(char)
			}
		}
	}
}

func (l *Lexer) Expect(tt TokenType, content string) error {
	token, err := l.Next()
	if err != nil {
		return err
	}

	if token.Type != tt {
		return token.Errorf("Expected token of type %v but found %v", tt, token.Type)
	}

	if content != "" && token.Content != content {
		return token.Errorf("Expected token of type %v with value %s but got %s", tt, content, token.Content)
	}

	return nil
}

func (l *Lexer) readToken() error {
	char, _, err := l.buffer.ReadRune()
	if err != nil {
		return err
	}
	l.col++

	switch char {
	case '#':
		return l.readHashLabel()
	case '$':
		return l.readSimpleLabel(DollarLabel)
	case '+':
		return l.readSimpleLabel(PlusLabel)
	case '"':
		return l.readString()
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.col--
		if err = l.buffer.UnreadRune(); err != nil {
			return err
		}

		return l.readNumber()
	case ';':
		return l.readLineComment()
	case '/':
		char, _, err = l.buffer.ReadRune()
		if err != nil {
			return err
		}

		if char == '*' {
			l.col++
			return l.readBlockComment()
		}
		l.buffer.UnreadRune()
		return l.errorf("Unrecognised token %s", string(char))

	case ' ', '\t', '\r', '\n':
		err = l.buffer.UnreadRune()
		if err != nil {
			return err
		}
		l.col--

		err := l.skipWhitespace()
		if err != nil {
			return err
		}
		return l.readToken()
	default:
		return l.errorf("Unrecognised token %s", string(char))
	}
}

func (l *Lexer) makeToken(tt TokenType) {
	l.next = &Token{
		Type:     tt,
		Content:  "",
		Location: [2]int{l.line + 1, l.col},
	}
}

func (l *Lexer) skipWhitespace() error {
	for {
		char, _, err := l.buffer.ReadRune()
		if err != nil {
			return err
		}

		if char == '\n' {
			l.line++
			l.col = 0
			continue
		}

		if char != ' ' && char != '\t' && char != '\r' {
			return l.buffer.UnreadRune()
		}

		l.col++
	}
}

func (l *Lexer) readUntil(stopchars string) (string, error) {
	result := ""
	for {
		char, _, err := l.buffer.ReadRune()
		if err != nil {
			if err == io.EOF && result != "" {
				return result, nil
			}

			return "", err
		}

		if strings.ContainsRune(stopchars, char) {
			err = l.buffer.UnreadRune()
			if err != nil {
				return "", err
			}
			return result, nil
		}

		if char == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}
		result += string(char)
	}
}

func (l *Lexer) requireRune(r rune) error {
	char, _, err := l.buffer.ReadRune()
	if err != nil {
		return err
	}

	if char == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}

	if char != r {
		return l.errorf("Expected '%s' but found '%s'", string(r), string(char))
	}

	return nil
}

func (l *Lexer) optionalRune(r rune) (bool, error) {
	char, _, err := l.buffer.ReadRune()
	if err != nil {
		return false, err
	}

	if char != r {
		if err = l.buffer.UnreadRune(); err != nil {
			return false, err
		}

		return false, nil
	}

	if char == '\n' {
		l.line++
		l.col = 0
	} else {
		l.col++
	}

	return true, nil
}

func (l *Lexer) readHashLabel() error {
	l.makeToken(HashLabel)
	label, err := l.readUntil("\t\n:;")
	if err != nil {
		l.next = nil
		return err
	}

	label = strings.Trim(label, " ")
	if label == "End" {
		l.next.Type = HashEnd
	}
	l.next.Content = label

	_, err = l.optionalRune(':')
	return err
}

func (l *Lexer) readSimpleLabel(tt TokenType) error {
	l.makeToken(tt)
	label, err := l.readUntil("\t\n:;")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = label
	_, err = l.optionalRune(':')
	return err
}

func (l *Lexer) readLineComment() error {
	l.makeToken(Comment)
	content, err := l.readUntil("\n")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = content
	return nil
}

func (l *Lexer) readBlockComment() error {
	l.makeToken(BlockComment)
	result := ""

	for {
		content, err := l.readUntil("*")
		if err != nil {
			l.next = nil
			return err
		}

		result += content

		// Skip the asterisk
		_, _, err = l.buffer.ReadRune()
		if err != nil {
			l.next = nil
			return err
		}
		l.col++

		char, _, err := l.buffer.ReadRune()
		if err != nil {
			l.next = nil
			return err
		}
		if char == '/' {
			l.col++
			break
		}

		result += "*"
		err = l.buffer.UnreadRune()
		if err != nil {
			l.next = nil
			return err
		}
	}

	l.next.Content = result
	return nil
}

func (l *Lexer) readLine() error {
	err := l.skipWhitespace()
	if err != nil {
		return err
	}

	l.makeToken(Line)
	content, err := l.readUntil("\n\t")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = strings.Trim(content, " ")
	return nil
}

func (l *Lexer) readWord() error {
	err := l.skipWhitespace()
	if err != nil {
		return err
	}

	l.makeToken(String)
	content, err := l.readUntil(",\n\t ")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = content
	return nil
}

func (l *Lexer) readString() error {
	l.makeToken(String)
	content, err := l.readUntil("\"")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = content
	if err = l.requireRune('"'); err != nil {
		l.next = nil
		return err
	}

	return nil
}

func (l *Lexer) readNumber() error {
	l.makeToken(Number)
	content, err := l.readUntil("(),\" \n\t")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = content
	return nil
}

func (l *Lexer) ReadList(cb func() error) error {
	if l.next != nil {
		return eris.New("Can't parse a list if another token has already been queued")
	}

	var err error
	if err = l.skipWhitespace(); err != nil {
		return err
	}

	if err = l.requireRune('('); err != nil {
		return err
	}

	for {
		if err = l.skipWhitespace(); err != nil {
			return err
		}

		char, _, err := l.buffer.ReadRune()
		if err != nil {
			return err
		}

		if char == ')' {
			l.col++
			break
		}

		if char == ',' {
			l.col++
			continue
		}

		if err = l.buffer.UnreadRune(); err != nil {
			return err
		}

		if err = cb(); err != nil {
			return err
		}
	}

	if err = l.skipWhitespace(); err != nil {
		return err
	}

	return nil
}
