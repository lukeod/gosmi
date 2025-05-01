package parser

import (
	"fmt"
	"strconv"

	"github.com/alecthomas/participle/v2/lexer"
	gosmilexer "github.com/lukeod/gosmi/parser/lexer"
	"github.com/lukeod/gosmi/types"
	// No need to import the token package directly here
)

type SubIdentifier struct {
	Pos    lexer.Position
	Name   *types.SmiIdentifier
	Number *types.SmiSubId
}

func (x *SubIdentifier) Parse(lex *lexer.PeekingLexer) error {
	peekedToken := lex.Peek()
	if peekedToken.EOF() {
		return fmt.Errorf("unexpected EOF at start of SubIdentifier parse")
	}

	// Get symbols from the refactored lexer definition
	symbols := (&gosmilexer.LexerDefinition{}).Symbols()
	intType := symbols["Int"]
	identType := symbols["Ident"]

	// Case 1: Just a number
	if peekedToken.Type == intType {
		token := lex.Next() // Consume the Int token
		x.Pos = token.Pos
		n, err := strconv.ParseUint(token.Value, 10, 32)
		if err != nil {
			return fmt.Errorf("Parse number: %w", err)
		}
		x.Number = new(types.SmiSubId)
		*x.Number = types.SmiSubId(n)
		return nil
	}

	// Case 2: Starts with an Identifier
	if peekedToken.Type == identType {
		identToken := lex.Next() // Consume the Ident token
		x.Pos = identToken.Pos
		x.Name = new(types.SmiIdentifier)
		*x.Name = types.SmiIdentifier(identToken.Value)

		// Peek ahead for optional "(Number)"
		peekParen := lex.Peek()
		if peekParen.EOF() || peekParen.Value != "(" {
			// It's just an identifier, parsing is complete
			return nil
		}

		// Consume "("
		lex.Next()

		// Consume Number
		numToken := lex.Next()
		if numToken.EOF() || numToken.Type != intType {
			return fmt.Errorf("unexpected %q after '(', expected Int", numToken)
		}
		n, err := strconv.ParseUint(numToken.Value, 10, 32)
		if err != nil {
			return fmt.Errorf("Parse number: %w", err)
		}
		x.Number = new(types.SmiSubId)
		*x.Number = types.SmiSubId(n)

		// Consume ")"
		closeParenToken := lex.Next()
		if closeParenToken.EOF() || closeParenToken.Value != ")" {
			return fmt.Errorf("unexpected %q after number %d, expected ')'", closeParenToken, n)
		}
		return nil
	}

	// If neither Int nor Ident, it's an error
	return fmt.Errorf("unexpected %q, expected Int or Ident", peekedToken)
}

type Oid struct {
	Pos lexer.Position

	SubIdentifiers []SubIdentifier `parser:"@@+"`
}

// Per RFC2578 Appendix A, not all valid ASN.1 refinements are allowed by SMI
// Specifically, MIN and MAX are not valid range values, nor is '<' permitted on the lower or upper end point
type Range struct {
	Pos lexer.Position

	Start string `parser:"@( \"-\"? Int | BinString | HexString | Ident )"`             // Allow Ident (for MIN/MAX)
	End   string `parser:"( \"..\" @( \"-\"? Int | BinString | HexString | Ident ) )?"` // Allow Ident (for MIN/MAX)
}

type Status string

const (
	StatusMandatory  Status = "mandatory"
	StatusOptional   Status = "optional"
	StatusCurrent    Status = "current"
	StatusDeprecated Status = "deprecated"
	StatusObsolete   Status = "obsolete"
)

func (s Status) ToSmi() types.Status {
	switch s {
	case StatusMandatory:
		return types.StatusMandatory
	case StatusOptional:
		return types.StatusOptional
	case StatusCurrent:
		return types.StatusCurrent
	case StatusDeprecated:
		return types.StatusDeprecated
	case StatusObsolete:
		return types.StatusObsolete
	}
	return types.StatusUnknown
}

type SubType struct {
	Pos lexer.Position

	OctetString []Range `parser:"( ( \"SIZE\" \"(\" ( @@ ( \"|\" @@ )* ) \")\" )"`
	Integer     []Range `parser:"| @@ ( \"|\" @@ )* )"`
}

type NamedNumber struct {
	Pos lexer.Position

	Name  types.SmiIdentifier `parser:"@Ident"`
	Value string              `parser:"\"(\" @( \"-\"? Int ) \")\""`
}

type SyntaxType struct {
	Pos lexer.Position

	Name    types.SmiIdentifier `parser:"@( OctetString | ObjectIdentifier | Ident )"`
	SubType *SubType            `parser:"( ( \"(\" @@ \")\" )"`
	Enum    []NamedNumber       `parser:"| ( \"{\" @@ ( \",\" @@ )* \",\"? \"}\" ) )?"`
}

type Syntax struct {
	Pos lexer.Position

	Sequence *types.SmiIdentifier `parser:"( \"SEQUENCE\" \"OF\" @Ident )"`
	Type     *SyntaxType          `parser:"| @@"`
}
