package parser

import (
	"context"
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

func (t Token) Range() [4]int {
	start := t.Location
	lines := strings.Split(t.Content, "\n")
	chars := start[1] + len(t.Content)

	if len(lines) > 1 {
		chars = len(lines[len(lines)-1])
	}
	return [4]int{start[0], start[1], start[0] + len(lines) - 1, chars}
}

func (t Token) Errorf(msg string, args ...interface{}) error {
	codeRange := t.Range()
	return eris.Wrap(NewParserError(fmt.Sprintf(msg, args...), codeRange), "")
}

func (t Token) GetLabel() string {
	switch t.Type {
	case HashLabel:
		return "#" + t.Content
	case DollarLabel:
		return "$" + t.Content
	case PlusLabel:
		return "+" + t.Content
	default:
		return ""
	}
}

type Scanner interface {
	io.RuneScanner
	io.Seeker
}

type savedPos struct {
	stream int64
	line   int
	col    int
}

type ScopeInfo struct {
	HoverText string
	Start     [2]int
	End       [2]int
}

type Lexer struct {
	ctx        context.Context
	buffer     Scanner
	next       *Token
	errors     []error
	warnings   []error
	scopeInfos []ScopeInfo
	posStack   []savedPos
	line       int
	col        int
}

func NewLexer(ctx context.Context, buffer Scanner) *Lexer {
	return &Lexer{ctx: ctx, buffer: buffer}
}

func (l *Lexer) errorf(msg string, args ...interface{}) error {
	return eris.Wrap(NewParserError(fmt.Sprintf(msg, args...), [4]int{l.line + 1, l.col - 1, l.line + 1, l.col}), "")
}

func (l *Lexer) addScopeInfo(token Token, info ScopeInfo) {
	codeRange := token.Range()
	info.Start = [2]int{codeRange[0], codeRange[1]}
	info.End = [2]int{codeRange[2], codeRange[3]}

	l.scopeInfos = append(l.scopeInfos, info)
}

func (l *Lexer) Errors() []error {
	return l.errors
}

func (l *Lexer) Warnings() []error {
	return l.warnings
}

func (l *Lexer) ScopeInfos() []ScopeInfo {
	return l.scopeInfos
}

func (l *Lexer) Report(e error) {
	l.errors = append(l.errors, e)
}

func (l *Lexer) ReportWarning(e error) {
	l.warnings = append(l.warnings, e)
}

func (l *Lexer) PushPosition() {
	streamPos, err := l.buffer.Seek(0, io.SeekCurrent)
	if err != nil {
		l.Report(err)
		return
	}

	l.posStack = append(l.posStack, savedPos{
		stream: streamPos,
		line:   l.line,
		col:    l.col,
	})
}

func (l *Lexer) PopPosition() {
	stackSize := len(l.posStack)
	frame := l.posStack[stackSize-1]
	l.posStack = l.posStack[:stackSize-1]

	_, err := l.buffer.Seek(frame.stream, io.SeekStart)
	if err != nil {
		l.Report(err)
	}
	l.line = frame.line
	l.col = frame.col
}

func (l *Lexer) DropPosition() {
	stackSize := len(l.posStack)
	l.posStack = l.posStack[:stackSize-1]
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

func (l *Lexer) ReadMultilineText(end string) (string, error) {
	result := make([]rune, 0, 200)
	if l.next != nil {
		result = append(result, []rune(l.next.Content)...)
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
					return string(result), nil
				}
			} else {
				result = append(result, []rune(end[:endpos])...)
				endpos = -1
			}
		} else {
			if char == '$' {
				endpos = 1
			} else {
				result = append(result, char)
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
	if l.ctx.Err() != nil {
		return l.ctx.Err()
	}

	l.PushPosition()
	char, _, err := l.buffer.ReadRune()
	if err != nil {
		return err
	}
	l.col++

	switch char {
	case '#':
		err = l.readHashLabel()
	case '$':
		err = l.readSimpleLabel(DollarLabel)
	case '+':
		err = l.readSimpleLabel(PlusLabel)
	case '"':
		err = l.readString()
	case '-', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		l.col--
		if err = l.buffer.UnreadRune(); err == nil {
			err = l.readNumber()
		}
	case ';':
		err = l.readLineComment()
	case '/':
		char, _, err = l.buffer.ReadRune()
		if err == nil {
			if char == '*' {
				l.col++
				err = l.readBlockComment()
			}
			if err == nil {
				l.buffer.UnreadRune()
				err = l.errorf("Unrecognised token %s", string(char))
			}
		}
	case ' ', '\t', '\r', '\n':
		err = l.buffer.UnreadRune()
		if err == nil {
			l.col--

			err = l.skipWhitespace()
			if err == nil {
				err = l.readToken()
			}
		}
	default:
		l.DropPosition()
		return l.errorf("Unrecognised token %s", string(char))
	}

	if err == nil {
		l.DropPosition()
	} else {
		l.PopPosition()
	}

	return err
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
	result := make([]rune, 0, 100)
	for {
		char, _, err := l.buffer.ReadRune()
		if err != nil {
			if err == io.EOF && len(result) > 0 {
				return string(result), nil
			}

			return "", err
		}

		if strings.ContainsRune(stopchars, char) {
			err = l.buffer.UnreadRune()
			if err != nil {
				return "", err
			}
			return string(result), nil
		}

		if char == '\n' {
			l.line++
			l.col = 0
		} else {
			l.col++
		}
		result = append(result, char)
	}
}

func (l *Lexer) readOnly(allowchars string) (string, error) {
	result := ""
	for {
		char, _, err := l.buffer.ReadRune()
		if err != nil {
			if err == io.EOF && result != "" {
				return result, nil
			}

			return "", err
		}

		if !strings.ContainsRune(allowchars, char) {
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
	label, err := l.readUntil("\r\t\n:;")
	if err != nil {
		l.next = nil
		return err
	}

	label = strings.Trim(label, " ")
	if label == "End" {
		l.next.Type = HashEnd
	}
	l.next.Content = label
	return nil
}

func (l *Lexer) readSimpleLabel(tt TokenType) error {
	l.makeToken(tt)
	label, err := l.readUntil("\r\t\n:;")
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
	l.makeToken(Line)
	content, err := l.readUntil(";\r\n")
	if err != nil {
		l.next = nil
		return err
	}

	l.next.Content = strings.Trim(content, " \t")
	return nil
}

func (l *Lexer) readWord() error {
	err := l.skipWhitespace()
	if err != nil {
		return err
	}

	l.makeToken(String)
	content, err := l.readUntil(",\r\n\t ")
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
	content, err := l.readOnly("0123456789.-")
	if err != nil {
		l.next = nil
		return err
	}

	if len(content) == 0 {
		return l.errorf("Exepcted a number")
	}

	if content[0] == '.' {
		content = "0" + content
	}
	l.next.Content = content
	return nil
}

func (l *Lexer) ReadList(cb func() error) error {
	if l.next != nil {
		return l.errorf("Can't parse a list if another token has already been queued")
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
