package token

import (
	"fmt"
	"strconv"

	"github.com/alecthomas/participle/v2/lexer"
)

// TokenType represents the type of token.
type TokenType int

// Token type constants. These correspond to the types expected by the Participle grammar.
// We use negative values to avoid collision with rune literals used by Participle's default lexer
// if it were ever mixed, and to clearly distinguish our custom types.
const (
	EOF TokenType = iota - 1 // Use -1 for EOF as Participle uses 0
	ILLEGAL
	Comment // -- comment
	Whitespace

	// Literals
	Ident      // Identifier (keywords are also initially lexed as Ident)
	Int        // Integer literal
	Text       // Double-quoted string "..."
	HexString  // Hex string '...'H
	BinString  // Binary string '...'B
	ExtUTCTime // Extended UTC Time string "..."
	ASN1Tag    // ASN.1 Tag [APPLICATION n]

	// Multi-word Keywords (lexed as single tokens)
	ObjectIdentifier // OBJECT IDENTIFIER
	OctetString      // OCTET STRING

	// Punctuation & Operators
	Assign    // ::=
	Range     // ..
	Pipe      // |
	LBrace    // {
	RBrace    // }
	LPAREN    // (
	RPAREN    // )
	Comma     // ,
	Semicolon // ;
	Dot       // .
	Minus     // -
	LBracket  // [ // Needed for ASN1Tag start
	RBracket  // ] // Needed for ASN1Tag end
)

// Token represents a lexed token.
type Token struct {
	Type  TokenType
	Value string
	Pos   lexer.Position
}

// String returns a string representation of the token.
func (t Token) String() string {
	val := t.Value
	if len(val) > 20 {
		val = val[:17] + "..."
	}
	return fmt.Sprintf("%s: %q (%s)", t.Pos, val, TokenTypeString(t.Type))
}

// TokenTypeString returns a string representation of the TokenType.
func TokenTypeString(tt TokenType) string {
	switch tt {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case Comment:
		return "Comment"
	case Whitespace:
		return "Whitespace"
	case Ident:
		return "Ident"
	case Int:
		return "Int"
	case Text:
		return "Text"
	case HexString:
		return "HexString"
	case BinString:
		return "BinString"
	case ExtUTCTime:
		return "ExtUTCTime"
	case ASN1Tag:
		return "ASN1Tag"
	case ObjectIdentifier:
		return "ObjectIdentifier"
	case OctetString:
		return "OctetString"
	case Assign:
		return "Assign"
	case Range:
		return "Range"
	case Pipe:
		return "Pipe"
	case LBrace:
		return "LBrace"
	case RBrace:
		return "RBrace"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case Comma:
		return "Comma"
	case Semicolon:
		return "Semicolon"
	case Dot:
		return "Dot"
	case Minus:
		return "Minus"
	case LBracket:
		return "LBracket"
	case RBracket:
		return "RBracket"
	default:
		return "Unknown(" + strconv.Itoa(int(tt)) + ")"
	}
}

// Symbols maps token type constants to their string representation for Participle.
// This helps Participle identify our custom token types.
var Symbols = map[TokenType]string{
	EOF:              "EOF",
	ILLEGAL:          "ILLEGAL",
	Comment:          "Comment",
	Whitespace:       "Whitespace",
	Ident:            "Ident",
	Int:              "Int",
	Text:             "Text",
	HexString:        "HexString",
	BinString:        "BinString",
	ExtUTCTime:       "ExtUTCTime",
	ASN1Tag:          "ASN1Tag",
	ObjectIdentifier: "ObjectIdentifier",
	OctetString:      "OctetString",
	Assign:           "Assign",
	Range:            "Range",
	Pipe:             "Pipe",
	LBrace:           "LBrace",
	RBrace:           "RBrace",
	LPAREN:           "LPAREN",
	RPAREN:           "RPAREN",
	Comma:            "Comma",
	Semicolon:        "Semicolon",
	Dot:              "Dot",
	Minus:            "Minus",
	LBracket:         "LBracket",
	RBracket:         "RBracket",
}
