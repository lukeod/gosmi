package parser

import (
	"fmt"
	"strconv"

	"github.com/alecthomas/participle/v2/lexer"

	"github.com/sleepinggenius2/gosmi/types"
)

type SubIdentifier struct {
	Pos    lexer.Position
	Name   *types.SmiIdentifier
	Number *types.SmiSubId
}

func (x *SubIdentifier) Parse(lex *lexer.PeekingLexer) error {
	token := lex.Next()
	if token.EOF() {
		return fmt.Errorf("unexpected EOF at start of SubIdentifier parse")
	}
	x.Pos = token.Pos
	symbols := smiLexer.Symbols()
	intType := symbols["Int"]
	identType := symbols["Ident"]
	if token.Type == intType {
		n, err := strconv.ParseUint(token.Value, 10, 32)
		if err != nil {
			return fmt.Errorf("Parse number: %w", err)
		}
		x.Number = new(types.SmiSubId)
		*x.Number = types.SmiSubId(n)
		return nil
	} else if token.Type != identType {
		return fmt.Errorf("Unexpected %q, expected Ident", token)
	}
	x.Name = new(types.SmiIdentifier)
	*x.Name = types.SmiIdentifier(token.Value)
	// Peek ahead to check for the optional number in parentheses
	peekedToken := lex.Peek()
	if peekedToken.EOF() {
		// It's okay to EOF after just an identifier name
		return nil
	}
	// If the next token is not "(", we just have the identifier name
	if peekedToken.Value != "(" {
		return nil
	}

	// Consume the "("
	consumedOpenParen := lex.Next()
	if consumedOpenParen.EOF() {
		return fmt.Errorf("unexpected EOF after identifier %q, expected '('", *x.Name)
	}
	// We already peeked and confirmed it's "(", so no need to check value again unless for robustness

	// Consume the number inside the parentheses
	numToken := lex.Next()
	if numToken.EOF() {
		return fmt.Errorf("unexpected EOF after '(', expected Int")
	}
	if numToken.Type != intType {
		return fmt.Errorf("Unexpected %q, expected Int", numToken)
	}
	n, err := strconv.ParseUint(numToken.Value, 10, 32)
	if err != nil {
		return fmt.Errorf("Parse number: %w", err)
	}
	x.Number = new(types.SmiSubId)
	*x.Number = types.SmiSubId(n)

	// Consume the ")"
	closeParenToken := lex.Next()
	if closeParenToken.EOF() {
		return fmt.Errorf("unexpected EOF after number %d, expected ')'", n)
	}
	if closeParenToken.Value != ")" {
		return fmt.Errorf("Unexpected %q, expected \")\"", closeParenToken)
	}
	return nil
}

type Oid struct {
	Pos lexer.Position

	SubIdentifiers []SubIdentifier `parser:"@@+"`
}

// Per RFC2578 Appendix A, not all valid ASN.1 refinements are allowed by SMI
// Specifically, MIN and MAX are not valid range values, nor is '<' permitted on the lower or upper end point
type Range struct {
	Pos lexer.Position

	Start string `parser:"@( \"-\"? Int | BinString | HexString )"`
	End   string `parser:"( \"..\" @( \"-\"? Int | BinString | HexString ) )?"`
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
