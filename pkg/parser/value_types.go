package parser

import (
	"strconv"
	"strings"
)

type SwitchItem struct {
	Items []ContainerChild
}

var _ ContainerChild = (*SwitchItem)(nil)

func (i SwitchItem) GetNames() []string {
	names := make([]string, 0, len(i.Items))
	for _, item := range i.Items {
		names = append(names, item.GetNames()...)
	}

	return names
}

func (i *SwitchItem) Parse(lex *Lexer) (interface{}, error) {
	var value interface{}
	var err error

	for _, item := range i.Items {
		value, err = item.Parse(lex)
		if err == nil && value != nil {
			return value, nil
		}
	}

	return nil, err
}

type ValueList struct {
	ValueParser ParseItem
}

var _ ParseItem = (*ValueList)(nil)

func (i ValueList) Parse(lex *Lexer) (interface{}, error) {
	result := make([]interface{}, 0)
	err := lex.ReadList(func() error {
		value, err := i.ValueParser.Parse(lex)
		if err != nil {
			return err
		}

		result = append(result, value)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

type FixedList struct {
	ValueParser ParseItem
	Size        int
}

var _ ParseItem = (*FixedList)(nil)

func (i FixedList) Parse(lex *Lexer) (interface{}, error) {
	result := make([]interface{}, i.Size)
	for idx := range result {
		value, err := i.ValueParser.Parse(lex)
		if err != nil {
			return nil, err
		}

		result[idx] = value
	}

	return result, nil
}

type (
	parseHandler     func(*Lexer) (interface{}, error)
	genericValueType struct {
		handler parseHandler
	}
)

var _ ParseItem = (*genericValueType)(nil)

func (g genericValueType) Parse(lex *Lexer) (interface{}, error) {
	return g.handler(lex)
}

func newGenericValueType(handler parseHandler) genericValueType {
	return genericValueType{handler: handler}
}

func consumeValue(lex *Lexer) (Token, error) {
	lex.PushPosition()
	token, err := lex.Next()
	if err != nil {
		return Token{}, err
	}

	if token.Type != Line && token.Type != Number && token.Type != String {
		lex.PopPosition()
		return Token{}, token.Errorf("Expected value but got %v", token.Type)
	}

	lex.DropPosition()
	return token, nil
}

var StringValue = newGenericValueType(func(lex *Lexer) (interface{}, error) {
	// Force the lexer to read a line
	err := lex.readLine()
	if err != nil {
		return nil, err
	}

	token, err := consumeValue(lex)
	if err != nil {
		return nil, err
	}

	result := strings.Trim(token.Content, " \n\t")
	// TODO: Parse XSTR?
	return result, nil
})

var StringFlag = newGenericValueType(func(lex *Lexer) (interface{}, error) {
	// Force the lexer to read a string
	err := lex.readString()
	if err != nil {
		return nil, err
	}

	token, err := consumeValue(lex)
	if err != nil {
		return nil, err
	}

	result := strings.Trim(token.Content, " \n\t")
	return result, nil
})

var WordValue = newGenericValueType(func(lex *Lexer) (interface{}, error) {
	// Force the lexer to read a Word
	err := lex.readWord()
	if err != nil {
		return nil, err
	}

	token, err := consumeValue(lex)
	if err != nil {
		return nil, err
	}

	return token.Content, nil
})

var MultilineStringValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	result, err := l.ReadMultilineText("$end_multi_text")
	if err != nil {
		return nil, err
	}

	return strings.Trim(result, " \n\t"), nil
})

var BooleanValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	// Force the lexer to read a word
	err := l.readWord()
	if err != nil {
		return nil, err
	}

	token, err := consumeValue(l)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(token.Content) {
	case "yes", "true", "ja", "oui", "si", "ita vero", "hija'", "hislah":
		return true, nil
	case "no", "false", "nein", "non", "minime", "ghobe'":
		return false, nil
	default:
		return nil, token.Errorf("Expected boolean but found %s", token.Content)
	}
})

var FloatValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	if err := l.skipWhitespace(); err != nil {
		return nil, err
	}

	// Force lexer to read a number
	if err := l.readNumber(); err != nil {
		return nil, err
	}

	token, err := consumeValue(l)
	if err != nil {
		return nil, err
	}

	value, err := strconv.ParseFloat(token.Content, 64)
	if err != nil {
		return nil, token.Errorf("Not a float: %s (%v)", token.Content, err)
	}

	return value, nil
})

var IntegerValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	if err := l.skipWhitespace(); err != nil {
		return nil, err
	}

	// Force lexer to read a number
	if err := l.readNumber(); err != nil {
		return nil, err
	}

	token, err := consumeValue(l)
	if err != nil {
		return nil, err
	}

	value, err := strconv.Atoi(token.Content)
	if err != nil {
		return nil, token.Errorf("Not an integer: %s (%v)", token.Content, err)
	}

	return value, nil
})

var FlagValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	// If we've come this far, the flag is present.
	return true, nil
})

var Vec3dValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	result := []float64{0, 0, 0}
	for idx := range result {
		if err := l.skipWhitespace(); err != nil {
			return nil, err
		}

		if _, err := l.optionalRune(','); err != nil {
			return nil, err
		}

		// Force the lexer to read a word
		err := l.readWord()
		if err != nil {
			return nil, err
		}

		token, err := l.Next()
		if err != nil {
			return nil, err
		}

		value, err := strconv.ParseFloat(token.Content, 64)
		if err != nil {
			return nil, token.Errorf("Failed to parse float %s (%v)", token.Content, err)
		}

		result[idx] = value
	}
	return result, nil
})

var ColorValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	// Force the lexer to read a line
	err := l.readLine()
	if err != nil {
		return nil, err
	}

	token, err := consumeValue(l)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(token.Content, " ")
	if len(parts) != 3 {
		return nil, token.Errorf("Expected vec3d but found %d parts", len(parts))
	}

	a, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, token.Errorf("Failed to parse int %s (%v)", parts[0], err)
	}

	b, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, token.Errorf("Failed to parse int %s (%v)", parts[1], err)
	}

	c, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, token.Errorf("Failed to parse int %s (%v)", parts[2], err)
	}

	if a > 255 || a < 0 || b > 255 || b < 0 || c > 255 || c < 0 {
		return nil, token.Errorf("One of these values is outside the valid range of 0-255: %d %d %d", a, b, c)
	}

	return []int{a, b, c}, nil
})

type Subsystem struct {
	Name       string
	HitPercent float64
	TurnRate   float64
}

var SubsystemValue = newGenericValueType(func(l *Lexer) (interface{}, error) {
	data, err := l.readUntil(",\n")
	if err != nil {
		return nil, err
	}

	result := Subsystem{
		Name: strings.Trim(data, " \t"),
	}

	found, err := l.optionalRune(',')
	if err != nil {
		return nil, err
	}

	if !found {
		return result, nil
	}

	l.PushPosition()
	token, err := l.Next()
	if err != nil {
		return nil, err
	}

	if token.Type != Number {
		l.PopPosition()
		return result, nil
	}

	l.DropPosition()
	result.HitPercent, err = strconv.ParseFloat(token.Content, 64)
	if err != nil {
		return nil, token.Errorf("Failed to parse hit percent %s (%s)", token.Content, err)
	}

	found, err = l.optionalRune(',')
	if err != nil {
		return nil, err
	}

	if !found {
		return result, nil
	}

	l.PushPosition()
	token, err = l.Next()
	if err != nil {
		return nil, err
	}

	if token.Type != Number {
		l.PopPosition()
		return result, nil
	}
	l.DropPosition()

	result.TurnRate, err = strconv.ParseFloat(token.Content, 64)
	if err != nil {
		return nil, token.Errorf("Failed to parse turn rate %s (%s)", token.Content, err)
	}

	return result, nil
})

var WeaponBankList = newGenericValueType(func(l *Lexer) (interface{}, error) {
	banks := make([][]string, 0, 2)
	for {
		if err := l.skipWhitespace(); err != nil {
			return nil, err
		}

		found, err := l.optionalRune('(')
		if err != nil {
			return nil, err
		}

		if !found {
			break
		}

		b := make([]string, 0)
		for {
			if err := l.skipWhitespace(); err != nil {
				return nil, err
			}

			found, err := l.optionalRune('"')
			if err != nil {
				return nil, err
			}

			if !found {
				break
			}

			value, err := l.readUntil("\"")
			if err != nil {
				return nil, err
			}

			b = append(b, value)

			if err = l.requireRune('"'); err != nil {
				return nil, err
			}
		}

		banks = append(banks, b)
		if err = l.requireRune(')'); err != nil {
			return nil, err
		}
	}

	return banks, nil
})
