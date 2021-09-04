package parser

import (
	"strings"

	"github.com/rotisserie/eris"
)

type ParseItem interface {
	Parse(lex *Lexer) (interface{}, error)
}

type ContainerChild interface {
	ParseItem
	GetNames() []string
}

type ContainerItem struct {
	Value             ParseItem
	Name              string
	DeprecatedMessage string
	Properties        []ContainerChild
	Multi             bool
	Required          bool
	BooleanContainer  bool
}

var _ ParseItem = (*ContainerItem)(nil)

func (c ContainerItem) GetNames() []string { return []string{c.Name} }

func (c ContainerItem) Parse(lex *Lexer) (interface{}, error) {
	if lex.ctx.Err() != nil {
		return nil, lex.ctx.Err()
	}

	if c.Multi {
		result := make([]interface{}, 0)
		required := c.Required
		for {
			item, err := c.ParseOne(lex, required)
			if err != nil {
				return nil, err
			}

			if item == nil {
				break
			}

			result = append(result, item)
			// Even if the container is required, any item after the first
			// is optional.
			required = false
		}

		return result, nil
	}

	return c.ParseOne(lex, c.Required)
}

func (c ContainerItem) ParseOne(lex *Lexer, required bool) (interface{}, error) {
	if c.Name == "" {
		if c.Value == nil {
			return nil, eris.Errorf("Encountered value item without value type %+v", c)
		}

		return c.Value.Parse(lex)
	}

	lex.PushPosition()
	token, err := lex.Next()
	if err != nil {
		return nil, err
	}

	var tt TokenType
	switch c.Name[0] {
	case '#':
		tt = HashLabel
	case '$':
		tt = DollarLabel
	case '+':
		tt = PlusLabel
	default:
		return nil, eris.Errorf("Invalid container name %s", c.Name)
	}

	if required {
		if token.Type != tt {
			lex.PopPosition()
			return nil, token.Errorf("Unexpected token %v. Expected %v", token.Type, tt)
		}

		if !strings.EqualFold(token.Content, c.Name[1:]) {
			lex.PopPosition()
			return nil, token.Errorf("Unexpected label %s. Expected %s", token.Content, c.Name)
		}
	} else {
		if token.Type != tt || !strings.EqualFold(token.Content, c.Name[1:]) {
			lex.PopPosition()
			return nil, nil
		}

		if c.DeprecatedMessage != "" {
			lex.DropPosition()
			return nil, token.Errorf("%s", c.DeprecatedMessage)
		}
	}
	lex.DropPosition()

	if c.Value != nil {
		return c.Value.Parse(lex)
	}

	if c.BooleanContainer {
		enabled, err := BooleanValue.Parse(lex)
		if err != nil {
			return nil, err
		}

		if !enabled.(bool) {
			return nil, nil
		}
	}

	knownFields := make(map[string]bool)
	for _, prop := range c.Properties {
		for _, name := range prop.GetNames() {
			knownFields[strings.ToLower(name)] = true
		}
	}

	singlesSeen := make(map[string]bool)
	result := make(map[string]interface{})
	for _, prop := range c.Properties {
		var token Token
		lex.PushPosition()
		for {
			token, err = lex.Next()
			// spew.Dump(token)
			if err == nil && token.GetLabel() != "" && singlesSeen[token.GetLabel()] {
				lex.Report(token.Errorf("Duplicate property %s", token.GetLabel()))
				lex.readLine()
				lex.Next()
				lex.DropPosition()
				lex.PushPosition()
			} else {
				break
			}
		}
		lex.PopPosition()

		val, err := prop.Parse(lex)
		if err != nil {
			lex.Report(err)
			// If we're not at the start of a new line, skip the rest of the current line
			if lex.col > 0 {
				lex.readLine()
			}

			// Skip any fields
			continue
		}

		if val != nil {
			if _, isSlice := val.([]interface{}); !isSlice {
				singlesSeen[token.GetLabel()] = true
			}

			if c.DeprecatedMessage != "" {
				lex.Report(token.Errorf("%s", c.DeprecatedMessage))
			}

			/*lex.addScopeInfo(token, ScopeInfo{
				HoverText: fmt.Sprintf("%+v", val),
			})*/

			isZero := false
			switch val := val.(type) {
			case string:
				isZero = val == ""
			case []interface{}:
				isZero = len(val) == 0
			}

			if !isZero {
				result[token.GetLabel()] = val
			}
		}
	}

	if c.Name[0] == '#' {
		token, err := lex.Next()
		if err != nil {
			return nil, err
		}

		if token.Type != HashEnd {
			// spew.Dump(result)
			return nil, token.Errorf("Expected '#End' but found '%s'", token.Content)
		}
	}

	return result, nil
}
