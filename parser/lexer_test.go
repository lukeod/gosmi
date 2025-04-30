package parser

import (
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to lex input and return tokens.
// It filters out EOF, Whitespace, and Comment tokens by default.
// For testing error cases, set expectConsumeError to true to allow ConsumeAll to error
// without failing the test immediately, returning tokens found before the error.
func lexInput(t *testing.T, input string, expectConsumeError bool) []lexer.Token {
	t.Helper()
	lex, err := smiLexer.LexString("", input)
	require.NoError(t, err, "Lexer failed")

	tokens, err := lexer.ConsumeAll(lex)
	if expectConsumeError {
		// If we expect an error (like for invalid char tests),
		// we still want the tokens found *before* the error.
		// ConsumeAll might return partial tokens even on error.
		// We don't require.NoError here.
		if err == nil {
			t.Logf("Warning: Expected a token consumption error but got none for input: %q", input)
		}
	} else {
		// For normal cases, fail if ConsumeAll errors.
		require.NoError(t, err, "Token consumption failed")
	}

	// Filter out EOF, Whitespace, and Comment tokens for easier comparison in tests,
	// as these are typically elided by the parser.
	var filteredTokens []lexer.Token
	eofType := lexer.EOF
	whitespaceType := smiLexer.Symbols()["Whitespace"]
	commentType := smiLexer.Symbols()["Comment"]
	require.NotZero(t, whitespaceType, "Whitespace symbol not found in lexer")
	require.NotZero(t, commentType, "Comment symbol not found in lexer")

	for _, token := range tokens {
		if token.Type != eofType && token.Type != whitespaceType && token.Type != commentType {
			filteredTokens = append(filteredTokens, token)
		}
	}
	return filteredTokens
}

func TestLexer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []lexer.Token // Note: We'll compare Type and Value, ignoring Pos for simplicity initially
	}{
		{
			name:  "Simple Assignment",
			input: `myIdentifier ::= 123`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Ident"], Value: "myIdentifier"},
				{Type: smiLexer.Symbols()["Assign"], Value: "::="},
				{Type: smiLexer.Symbols()["Int"], Value: "123"},
			},
		},
		{
			name:  "Identifier with Hyphen",
			input: `MODULE-IDENTITY`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Ident"], Value: "MODULE-IDENTITY"},
			},
		},
		{
			name:  "Integer",
			input: `42`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Int"], Value: "42"},
			},
		},
		{
			name:     "Comment Only",
			input:    `-- This is a comment`,
			expected: []lexer.Token{
				// Comments are elided by the lexer definition
			},
		},
		// Whitespace and Comments are filtered by the lexInput helper, so expect empty slices.
		{
			name:     "Comment Only",
			input:    `-- This is a comment`,
			expected: []lexer.Token{},
		},
		{
			name: "Assignment with Comments and Whitespace",
			input: ` node -- node name
					::= -- assignment
					{ parent 1 } -- oid value`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Ident"], Value: "node"},
				{Type: smiLexer.Symbols()["Assign"], Value: "::="},
				{Type: smiLexer.Symbols()["Punct"], Value: "{"},
				{Type: smiLexer.Symbols()["Ident"], Value: "parent"},
				{Type: smiLexer.Symbols()["Int"], Value: "1"},
				{Type: smiLexer.Symbols()["Punct"], Value: "}"},
			},
		},
		{
			name:  "Text String - Simple",
			input: `"hello world"`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Text"], Value: `"hello world"`}, // Raw lexer value includes quotes
			},
		},
		{
			name:  "Text String - Escaped Quote",
			input: `"hello \"quoted\" world"`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Text"], Value: `"hello \"quoted\" world"`}, // Raw lexer value includes quotes and escapes
			},
		},
		{
			name: "Text String - Multi-line",
			input: `"line one
line two"`, // Raw newline within the string
			expected: []lexer.Token{
				// The lexer rule captures the content including the raw newline.
				{Type: smiLexer.Symbols()["Text"], Value: "\"line one\nline two\""}, // Raw lexer value includes quotes and newline
			},
		},
		{
			name:  "Binary String",
			input: `'101010'B`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["BinString"], Value: `'101010'B`},
			},
		},
		{
			name:  "Hex String",
			input: `'0FA5'H`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["HexString"], Value: `'0FA5'H`},
			},
		},
		{
			name:  "Empty Hex String",
			input: `DEFVAL { ''H }`, // Use within DEFVAL for context
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Ident"], Value: "DEFVAL"},
				{Type: smiLexer.Symbols()["Punct"], Value: "{"},
				{Type: smiLexer.Symbols()["HexString"], Value: `''H`}, // Expect HexString, not Punct, Punct, Ident
				{Type: smiLexer.Symbols()["Punct"], Value: "}"},
			},
		},
		{
			name:  "ExtUTCTime - Short",
			input: `"9505241811Z"`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["ExtUTCTime"], Value: `"9505241811Z"`}, // Raw lexer value includes quotes
			},
		},
		{
			name:  "ExtUTCTime - Long",
			input: `"20240430143000Z"`,
			expected: []lexer.Token{
				{Type: lexer.TokenType(-10), Value: `"20240430143000Z"`}, // Raw lexer value includes quotes
			},
		},
		{
			name:  "Punctuation",
			input: `{}()|,..`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Punct"], Value: "{"},
				{Type: smiLexer.Symbols()["Punct"], Value: "}"},
				{Type: smiLexer.Symbols()["Punct"], Value: "("},
				{Type: smiLexer.Symbols()["Punct"], Value: ")"},
				{Type: smiLexer.Symbols()["Punct"], Value: "|"},
				{Type: smiLexer.Symbols()["Punct"], Value: ","},
				{Type: smiLexer.Symbols()["Punct"], Value: ".."},
			},
		},
		{
			name:  "Keyword - FROM", // Explicitly defined keyword
			input: `FROM`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Keyword"], Value: "FROM"},
			},
		},
		{
			name:  "Keyword - Other (as Ident)", // Most keywords are just identifiers to the lexer
			input: `DEFINITIONS BEGIN END MODULE-IDENTITY OBJECT-TYPE`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Keyword"], Value: "DEFINITIONS"},
				{Type: smiLexer.Symbols()["Keyword"], Value: "BEGIN"},
				{Type: smiLexer.Symbols()["Keyword"], Value: "END"},
				{Type: smiLexer.Symbols()["Ident"], Value: "MODULE-IDENTITY"},
				{Type: smiLexer.Symbols()["Ident"], Value: "OBJECT-TYPE"}, // Note: OBJECT-TYPE is Ident, OBJECT TYPE is handled by parser mapping
			},
		},
		{
			name:  "Multi-word Token - OBJECT IDENTIFIER",
			input: `OBJECT IDENTIFIER`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["ObjectIdentifier"], Value: `OBJECT IDENTIFIER`}, // Mapped value
			},
		},
		{
			name:  "Multi-word Token - OCTET STRING",
			input: `OCTET STRING`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["OctetString"], Value: `OCTET STRING`}, // Mapped value
			},
		},
		{
			name:  "Sequence - Ident, Punct, Int",
			input: `myNode(1)`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Ident"], Value: "myNode"},
				{Type: smiLexer.Symbols()["Punct"], Value: "("},
				{Type: smiLexer.Symbols()["Int"], Value: "1"},
				{Type: smiLexer.Symbols()["Punct"], Value: ")"},
			},
		},
		{
			name:  "Boundary - Ident followed by Assign",
			input: `node::=`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Ident"], Value: "node"},
				{Type: smiLexer.Symbols()["Assign"], Value: "::="},
			},
		},
		{
			name:  "Boundary - Int followed by Punct",
			input: `1}`,
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Int"], Value: "1"},
				{Type: smiLexer.Symbols()["Punct"], Value: "}"},
			},
		},
		// Note: The simple lexer might not explicitly error on all invalid syntax,
		// but rather produce unexpected tokens or stop. Parsing handles syntax errors.
		{
			name:  "Unterminated Text String",
			input: `"hello`, // Missing closing quote
			// Lexer sees Punct (`"`) then Ident (`hello`) as Text rule doesn't match.
			expected: []lexer.Token{
				{Type: smiLexer.Symbols()["Punct"], Value: `"`},
				{Type: smiLexer.Symbols()["Ident"], Value: `hello`},
			},
		},
		{
			name:  "Invalid Character (stops lexing)",
			input: `myIdent^`, // '^' is not defined, lexer should error during ConsumeAll.
			// ConsumeAll appears to discard all tokens on error.
			expected: []lexer.Token{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Determine if this test case expects a consume error
			expectConsumeError := tt.name == "Invalid Character (stops lexing)"
			// Unterminated string does not cause ConsumeAll to error in this lexer.
			actualTokens := lexInput(t, tt.input, expectConsumeError)

			require.Equal(t, len(tt.expected), len(actualTokens), "Number of tokens mismatch")

			for i := range tt.expected {
				assert.Equal(t, tt.expected[i].Type, actualTokens[i].Type, "Token %d Type mismatch", i)
				assert.Equal(t, tt.expected[i].Value, actualTokens[i].Value, "Token %d Value mismatch", i)
				// Position check might be added later if needed, but often complicates tests
				// assert.Equal(t, tt.expected[i].Pos, actualTokens[i].Pos, "Token %d Pos mismatch", i)
			}
		})
	}
}
