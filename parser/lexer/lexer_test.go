package lexer

import (
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/lukeod/gosmi/parser/lexer/token" // Corrected import path
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to lex input and return all non-whitespace/comment tokens until EOF.
// Uses the new lexer interface.
func lexAll(t *testing.T, input string) []lexer.Token {
	l := NewLexer("test.smi", input) // Provide a dummy filename
	var tokens []lexer.Token
	for {
		tok, err := l.Next()
		require.NoError(t, err, "Lexer Next() returned an unexpected error")

		// Skip whitespace and comments using the underlying token type values
		// Note: We cast our custom token types to lexer.TokenType for comparison
		if tok.Type != lexer.TokenType(token.Whitespace) && tok.Type != lexer.TokenType(token.Comment) {
			tokens = append(tokens, tok)
		}

		// Participle uses lexer.EOF (-1) for end of file
		if tok.Type == lexer.EOF {
			break
		}
		// Add a safeguard against infinite loops in case of lexer bugs
		if len(tokens) > 1000 {
			t.Fatal("Lexer produced too many tokens, possible infinite loop")
			break
		}
	}
	return tokens
}

func TestLexerBasicTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		// Expected still uses token.Token for convenience, but we compare against lexer.Token
		expected []token.Token
	}{
		{
			name:  "Simple Identifiers and Numbers",
			input: "TEST-MIB DEFINITIONS ::= BEGIN 123 END",
			expected: []token.Token{
				{Type: token.Ident, Value: "TEST-MIB"},
				{Type: token.Ident, Value: "DEFINITIONS"},
				{Type: token.Assign, Value: "::="},
				{Type: token.Ident, Value: "BEGIN"},
				{Type: token.Int, Value: "123"},
				{Type: token.Ident, Value: "END"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Punctuation",
			input: "{}(),;..|-",
			expected: []token.Token{
				{Type: token.LBrace, Value: "{"},
				{Type: token.RBrace, Value: "}"},
				{Type: token.LPAREN, Value: "("},
				{Type: token.RPAREN, Value: ")"},
				{Type: token.Comma, Value: ","},
				{Type: token.Semicolon, Value: ";"},
				{Type: token.Range, Value: ".."},
				{Type: token.Pipe, Value: "|"},
				{Type: token.Minus, Value: "-"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Dot",
			input: ".",
			expected: []token.Token{
				{Type: token.Dot, Value: "."},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Whitespace and Comments",
			input: "ident1 -- comment\n  ident2 -- another\n\tident3",
			expected: []token.Token{
				{Type: token.Ident, Value: "ident1"},
				{Type: token.Ident, Value: "ident2"},
				{Type: token.Ident, Value: "ident3"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Identifier immediately followed by comment",
			input: "ident1--comment\nident2 -- also comment",
			expected: []token.Token{
				{Type: token.Ident, Value: "ident1"}, // Should now be emitted
				{Type: token.Ident, Value: "ident2"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Empty Input",
			input: "",
			expected: []token.Token{
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Only Whitespace",
			input: "  \n\t \r ",
			expected: []token.Token{
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Only Comment",
			input: "-- this is a comment",
			expected: []token.Token{
				{Type: token.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTokens := lexAll(t, tt.input)

			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")

			for i := range tt.expected {
				// Compare expected custom type (cast to lexer.TokenType) with actual lexer.TokenType
				assert.Equal(t, lexer.TokenType(tt.expected[i].Type), actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
				// Basic position check: Ensure line numbers are generally increasing or same
				// Note: actualTokens[i] is now lexer.Token which has Pos field
				if i > 0 && actualTokens[i].Pos.Line < actualTokens[i-1].Pos.Line {
					t.Errorf("Token %d Line number decreased: %d -> %d", i, actualTokens[i-1].Pos.Line, actualTokens[i].Pos.Line)
				}
				// TODO: Add more detailed position checks if necessary
			}
		})
	}
}

func TestLexerPositionTracking(t *testing.T) {
	input := `LINE1
LINE2 IDENT
	 LINE3 -- comment
LINE4`
	filename := "pos_test.smi"
	l := NewLexer(filename, input) // Use new signature

	// Expected tokens are now lexer.Token with correct filename
	expectedTokens := []lexer.Token{
		{Type: lexer.TokenType(token.Ident), Value: "LINE1", Pos: lexer.Position{Filename: filename, Offset: 0, Line: 1, Column: 1}},
		{Type: lexer.TokenType(token.Ident), Value: "LINE2", Pos: lexer.Position{Filename: filename, Offset: 6, Line: 2, Column: 1}},
		{Type: lexer.TokenType(token.Ident), Value: "IDENT", Pos: lexer.Position{Filename: filename, Offset: 12, Line: 2, Column: 7}},
		{Type: lexer.TokenType(token.Ident), Value: "LINE3", Pos: lexer.Position{Filename: filename, Offset: 20, Line: 3, Column: 3}},
		// Comment is skipped
		{Type: lexer.TokenType(token.Ident), Value: "LINE4", Pos: lexer.Position{Filename: filename, Offset: 37, Line: 4, Column: 1}},
		{Type: lexer.EOF, Value: "", Pos: lexer.Position{Filename: filename, Offset: 42, Line: 4, Column: 6}}, // EOF type is lexer.EOF
	}

	var actualTokens []lexer.Token // Store lexer.Token
	for {
		tok, err := l.Next() // Use Next()
		require.NoError(t, err, "Lexer Next() returned an unexpected error")

		// Skip whitespace/comment
		if tok.Type == lexer.TokenType(token.Whitespace) || tok.Type == lexer.TokenType(token.Comment) {
			continue
		}
		actualTokens = append(actualTokens, tok)
		if tok.Type == lexer.EOF { // Check for lexer.EOF
			break
		}
	}

	require.Len(t, actualTokens, len(expectedTokens), "Number of tokens mismatch")

	for i, expectedTok := range expectedTokens {
		// Compare fields of lexer.Token directly
		assert.Equal(t, expectedTok.Type, actualTokens[i].Type, "Token %d Type mismatch", i)
		assert.Equal(t, expectedTok.Value, actualTokens[i].Value, "Token %d Value mismatch", i)
		// Compare the whole Position struct
		assert.Equal(t, expectedTok.Pos, actualTokens[i].Pos, "Token %d (%s) Position mismatch", i, token.TokenTypeString(token.TokenType(actualTokens[i].Type)))
	}
}

func TestLexerStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token.Token
	}{
		{
			name:  "Simple Text",
			input: `"hello world"`,
			expected: []token.Token{
				{Type: token.Text, Value: `"hello world"`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Text with escaped quote",
			input: `"hello \"quoted\" world"`,
			expected: []token.Token{
				{Type: token.Text, Value: `"hello \"quoted\" world"`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Multiline Text",
			input: "\"line one\nline two\"",
			expected: []token.Token{
				{Type: token.Text, Value: "\"line one\nline two\""},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Empty Text",
			input: `""`,
			expected: []token.Token{
				{Type: token.Text, Value: `""`},
				{Type: token.EOF, Value: ""},
			},
		},
		{ // Start of ExtUTCTime Long test case
			name:  "ExtUTCTime Long (YYYYMMDDHHMMZ)", // Corrected format
			input: `"202405011230Z"`,                 // Use 13 digits + Z
			expected: []token.Token{
				{Type: token.ExtUTCTime, Value: `"202405011230Z"`}, // Expected value updated
				{Type: token.EOF, Value: ""},
			},
		}, // End of ExtUTCTime Long test case
		{ // Start of ExtUTCTime Short test case
			name:  "ExtUTCTime Short",
			input: `"9505241811Z"`,
			expected: []token.Token{
				{Type: token.ExtUTCTime, Value: `"9505241811Z"`},
				{Type: token.EOF, Value: ""},
			},
		}, // End of ExtUTCTime Short test case
		{
			name:  "Text looks like UTC but wrong length",
			input: `"20240501Z"`,
			expected: []token.Token{
				{Type: token.Text, Value: `"20240501Z"`}, // Should be Text, not ExtUTCTime
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Text looks like UTC but no Z",
			input: `"20240501123000"`,
			expected: []token.Token{
				{Type: token.Text, Value: `"20240501123000"`}, // Should be Text
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Unterminated Text",
			input: `"abc`,
			expected: []token.Token{
				// Lexer currently emits ILLEGAL for unterminated strings
				{Type: token.ILLEGAL, Value: `"abc`},
				{Type: token.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTokens := lexAll(t, tt.input)
			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")
			for i := range tt.expected {
				assert.Equal(t, lexer.TokenType(tt.expected[i].Type), actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
			}
			// TODO: Check for recorded errors in error cases like unterminated string
		})
	}
}

func TestLexerHexBinStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token.Token
	}{
		{
			name:  "Simple HexString Upper H",
			input: `'0AF1'H`,
			expected: []token.Token{
				{Type: token.HexString, Value: `'0AF1'H`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Simple HexString Lower h",
			input: `'0af1'h`,
			expected: []token.Token{
				{Type: token.HexString, Value: `'0af1'h`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Empty HexString",
			input: `''H`,
			expected: []token.Token{
				{Type: token.HexString, Value: `''H`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Simple BinString Upper B",
			input: `'0110'B`,
			expected: []token.Token{
				{Type: token.BinString, Value: `'0110'B`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Simple BinString Lower b",
			input: `'101'b`,
			expected: []token.Token{
				{Type: token.BinString, Value: `'101'b`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Empty BinString",
			input: `''B`,
			expected: []token.Token{
				{Type: token.BinString, Value: `''B`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Invalid Hex Char",
			input: `'0AG'H`,
			expected: []token.Token{
				// Lexer currently emits ILLEGAL for invalid content/suffix
				{Type: token.ILLEGAL, Value: `'0AG'H`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Invalid Bin Char",
			input: `'012'B`,
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: `'012'B`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Unterminated Hex",
			input: `'0AF`,
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: `'0AF`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Missing Suffix",
			input: `'0AF'`,
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: `'0AF'`},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Wrong Suffix",
			input: `'0AF'X`,
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: `'0AF'X`},
				{Type: token.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTokens := lexAll(t, tt.input)
			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")
			for i := range tt.expected {
				assert.Equal(t, lexer.TokenType(tt.expected[i].Type), actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
			}
			// TODO: Check for recorded errors in error cases
		})
	}
}

func TestLexerMultiWordKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token.Token
	}{
		{
			name:  "OBJECT IDENTIFIER Simple",
			input: "OBJECT IDENTIFIER",
			expected: []token.Token{
				{Type: token.ObjectIdentifier, Value: "OBJECT IDENTIFIER"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OBJECT IDENTIFIER Extra Space",
			input: "OBJECT  IDENTIFIER",
			expected: []token.Token{
				{Type: token.ObjectIdentifier, Value: "OBJECT  IDENTIFIER"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OBJECT IDENTIFIER Newline",
			input: "OBJECT\nIDENTIFIER",
			expected: []token.Token{
				{Type: token.ObjectIdentifier, Value: "OBJECT\nIDENTIFIER"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OBJECT IDENTIFIER With Comment",
			input: "OBJECT -- comment\n IDENTIFIER",
			expected: []token.Token{
				{Type: token.ObjectIdentifier, Value: "OBJECT -- comment\n IDENTIFIER"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OCTET STRING Simple",
			input: "OCTET STRING",
			expected: []token.Token{
				{Type: token.OctetString, Value: "OCTET STRING"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OCTET STRING Extra Space",
			input: "OCTET   STRING",
			expected: []token.Token{
				{Type: token.OctetString, Value: "OCTET   STRING"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OCTET STRING Newline",
			input: "OCTET\nSTRING",
			expected: []token.Token{
				{Type: token.OctetString, Value: "OCTET\nSTRING"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OCTET STRING With Comment",
			input: "OCTET -- comment\n STRING",
			expected: []token.Token{
				{Type: token.OctetString, Value: "OCTET -- comment\n STRING"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OBJECT followed by non-IDENTIFIER",
			input: "OBJECT {",
			expected: []token.Token{
				{Type: token.Ident, Value: "OBJECT"}, // Should be lexed as Ident
				{Type: token.LBrace, Value: "{"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "OCTET followed by non-STRING",
			input: "OCTET 123",
			expected: []token.Token{
				{Type: token.Ident, Value: "OCTET"}, // Should be lexed as Ident
				{Type: token.Int, Value: "123"},
				{Type: token.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTokens := lexAll(t, tt.input)
			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")
			for i := range tt.expected {
				assert.Equal(t, lexer.TokenType(tt.expected[i].Type), actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
			}
		})
	}
}

func TestLexerASN1Tags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token.Token
	}{
		{
			name:  "Valid ASN1 Tag",
			input: "[APPLICATION 123]",
			expected: []token.Token{
				{Type: token.ASN1Tag, Value: "[APPLICATION 123]"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Valid ASN1 Tag with spaces",
			input: "[ APPLICATION   45 ]",
			expected: []token.Token{
				{Type: token.ASN1Tag, Value: "[ APPLICATION   45 ]"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Invalid ASN1 Tag - Missing APPLICATION",
			input: "[ 123 ]",
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: "["}, // Fails on "APPLICATION" check
				{Type: token.Int, Value: "123"},   // Continues lexing
				{Type: token.ILLEGAL, Value: "]"}, // ']' becomes ILLEGAL
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Invalid ASN1 Tag - Missing Digits",
			input: "[APPLICATION]",
			expected: []token.Token{
				// Lexer emits ILLEGAL up to the point where digits were expected
				{Type: token.ILLEGAL, Value: "[APPLICATION"}, // Emits illegal up to where digits expected
				{Type: token.ILLEGAL, Value: "]"},            // Then ']' becomes illegal
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Invalid ASN1 Tag - Missing Closing Bracket",
			input: "[APPLICATION 123",
			expected: []token.Token{
				// Lexer emits ILLEGAL up to the point where ']' was expected (EOF in this case)
				// The fix in lexer.go ensures it emits the whole invalid part.
				{Type: token.ILLEGAL, Value: "[APPLICATION 123"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Invalid ASN1 Tag - Wrong Keyword",
			input: "[APPLI 123]",
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: "["}, // Fails on "APPLICATION" check
				{Type: token.Ident, Value: "APPLI"},
				{Type: token.Int, Value: "123"},
				{Type: token.ILLEGAL, Value: "]"}, // Assuming ']' hits the default case
				{Type: token.EOF, Value: ""},      // Corrected trailing comma position
			},
		},
		{
			name:  "Just Brackets",
			input: "[]",
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: "["}, // Fails on "APPLICATION" check
				{Type: token.ILLEGAL, Value: "]"}, // Assuming ']' hits the default case
				{Type: token.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTokens := lexAll(t, tt.input)
			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")
			for i := range tt.expected {
				assert.Equal(t, lexer.TokenType(tt.expected[i].Type), actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
			}
		})
	}
}

func TestLexerIllegalChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token.Token
	}{
		{
			name:  "Single Illegal Character",
			input: "@",
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: "@"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Illegal Character within valid tokens",
			input: "ident # ident2",
			expected: []token.Token{
				{Type: token.Ident, Value: "ident"},
				{Type: token.ILLEGAL, Value: "#"},
				{Type: token.Ident, Value: "ident2"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Single Colon",
			input: ":",
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: ":"},
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Double Colon Not Assign",
			input: "::X",
			expected: []token.Token{
				// The fix in lexer.go ensures '::' is emitted as one ILLEGAL token
				{Type: token.ILLEGAL, Value: "::"},
				{Type: token.Ident, Value: "X"}, // Continues lexing
				{Type: token.EOF, Value: ""},
			},
		},
		{
			name:  "Double Colon At EOF",
			input: "::",
			expected: []token.Token{
				{Type: token.ILLEGAL, Value: "::"}, // Emits illegal for '::' if not followed by '='
				{Type: token.EOF, Value: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTokens := lexAll(t, tt.input)
			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")
			for i := range tt.expected {
				assert.Equal(t, lexer.TokenType(tt.expected[i].Type), actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
			}
		})
	}
}

// TODO: Add tests for:
// - Error cases (illegal characters, unterminated strings) - More specific error checks
// - Comment edge cases (EOF, identifier followed by comment)
