package parser

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"

	"github.com/lukeod/gosmi/types"
)

type MacroBody struct {
	Pos lexer.Position

	TypeNotation  string
	ValueNotation string
	Tokens        map[string]string
}

func (m *MacroBody) Parse(lex *lexer.PeekingLexer) error {
	token := lex.Next()
	if token.EOF() {
		return fmt.Errorf("unexpected EOF, expected 'BEGIN'")
	}
	if token.Value != "BEGIN" {
		return fmt.Errorf("Expected 'BEGIN', Got '%s'", token.Value)
	}
	m.Pos = token.Pos

	var tokenName, tokenValue string
	m.Tokens = make(map[string]string)
	symbols := smiLexer.Symbols()
	for {
		token = lex.Next()
		if token.EOF() {
			return fmt.Errorf("unexpected EOF, expected 'END'")
		}
		if token.Value == "END" {
			break
		}
		peek := lex.Peek()
		assignType := symbols["Assign"]
		textType := symbols["Text"]
		if ((token.Value == "TYPE" || token.Value == "VALUE") && peek.Value == "NOTATION") || peek.Type == assignType {
			if peek.Value == "NOTATION" {
				tokenName += " NOTATION"
				// Consume the peeked "NOTATION" token
				lex.Next()
				continue
			}
			if tokenName != "" {
				switch tokenName {
				case "TYPE NOTATION":
					m.TypeNotation = tokenValue
				case "VALUE NOTATION":
					m.ValueNotation = tokenValue
				default:
					m.Tokens[tokenName] = tokenValue
				}
			}
			tokenName = token.Value
			tokenValue = ""
			if peek.Type == assignType {
				// Consume the peeked "Assign" token
				lex.Next()
			}
			continue
		}
		if len(tokenValue) > 0 {
			tokenValue += " "
		}
		if token.Type == textType {
			tokenValue += `"` + token.Value + `"`
		} else {
			tokenValue += token.Value
		}
	}
	switch tokenName {
	case "":
		break
	case "TYPE NOTATION":
		m.TypeNotation = tokenValue
	case "VALUE NOTATION":
		m.ValueNotation = tokenValue
	default:
		m.Tokens[tokenName] = tokenValue
	}
	return nil
}

type Macro struct {
	Pos lexer.Position

	Name types.SmiIdentifier `parser:"@Ident \"MACRO\" Assign"`
	Body MacroBody           `parser:"@@"`
}
