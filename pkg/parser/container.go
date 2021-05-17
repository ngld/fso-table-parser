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
	GetName() string
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

func (c ContainerItem) GetName() string { return c.Name }

func (c ContainerItem) Parse(lex *Lexer) (interface{}, error) {
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

	token, err := lex.Peek()
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
			return nil, token.Errorf("Unexpected token %v. Expected %v", token.Type, tt)
		}

		if !strings.EqualFold(token.Content, c.Name[1:]) {
			return nil, token.Errorf("Unexpected label %s. Expected %s", token.Content, c.Name)
		}
	} else {
		if token.Type != tt || !strings.EqualFold(token.Content, c.Name[1:]) {
			return nil, nil
		}

		if c.DeprecatedMessage != "" {
			return nil, token.Errorf("%s", c.DeprecatedMessage)
		}
	}

	err = lex.Consume()
	if err != nil {
		return nil, err
	}

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

	result := make(map[string]interface{})
	for _, prop := range c.Properties {
		val, err := prop.Parse(lex)
		if err != nil {
			return nil, err
		}

		result[prop.GetName()] = val
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
