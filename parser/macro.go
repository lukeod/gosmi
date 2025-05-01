package parser

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"

	gosmilexer "github.com/lukeod/gosmi/parser/lexer" // Import the refactored lexer package
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
	// Get symbols from the refactored lexer definition
	symbols := (&gosmilexer.LexerDefinition{}).Symbols()
	assignType := symbols["Assign"]
	textType := symbols["Text"]

	// Initial token fetch was done before the loop starts (line 21)
	for {
		if token.EOF() {
			return fmt.Errorf("unexpected EOF, expected 'END'")
		}
		if token.Value == "END" {
			break // Exit loop when END is encountered
		}

		peek := lex.Peek()

		// Check if the current token/peek indicates the start of a new section
		isNotationStart := (token.Value == "TYPE" || token.Value == "VALUE") && peek.Value == "NOTATION"
		isAssignStart := peek.Type == assignType

		if isNotationStart || isAssignStart {
			// Assign the previously accumulated name/value before starting the new section
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

			// Set up the new section
			if isNotationStart {
				tokenName = token.Value + " NOTATION"
				tokenValue = ""
				lex.Next()         // Consume NOTATION
				token = lex.Next() // Fetch the token *after* NOTATION (should be ::=)
			} else { // isAssignStart
				tokenName = token.Value
				tokenValue = ""
				lex.Next()         // Consume ::=
				token = lex.Next() // Fetch the token *after* ::=
			}
			// After consuming the delimiter and fetching the next token, restart the loop
			// This avoids appending the delimiter (::= or NOTATION) or the section name token
			continue
		}

		// If not a section start, append the current token's value
		if len(tokenValue) > 0 {
			tokenValue += " "
		}
		if token.Type == textType {
			// Re-add quotes for Text tokens as the lexer strips them
			tokenValue += `"` + token.Value + `"`
		} else {
			tokenValue += token.Value
		}

		// Fetch the next token for the next iteration
		token = lex.Next()
	}

	// Assign the last accumulated section after the loop finishes
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
