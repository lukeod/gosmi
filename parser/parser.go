package parser

import (
	"fmt"
	"io"
	"os"
	"regexp"  // For strconv.Unquote (potentially)
	"strings" // For strings.ReplaceAll

	"github.com/alecthomas/participle/v2"       // Updated import path
	"github.com/alecthomas/participle/v2/lexer" // Updated import path
	// Removed ebnf import: "github.com/alecthomas/participle/lexer/ebnf"
)

var (
	// Define the lexer using lexer.NewSimple
	smiLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"Comment", `--[^\n]*`},
		{"Whitespace", `[ \t\n\r]+`},
		// Keywords and specific multi-word tokens need to be defined before Ident
		// Use non-capturing groups for spaces to avoid them being part of the token value if needed,
		// although participle.Map is used later anyway.
		{"ObjectIdentifier", `OBJECT\s+IDENTIFIER`},
		{"OctetString", `OCTET\s+STRING`},
		{"Keyword", `FROM`}, // Example keyword, others handled by Ident + parser rules
		{"Assign", `::=`},
		{"ExtUTCTime", `"(\d{10}(\d{2})?[zZ])"`}, // Capture content inside quotes
		{"Text", `"(\\.|[^"])*"`},                // Capture content inside quotes, allowing escaped quotes
		{"BinString", `'[01]+'[bB]`},
		{"HexString", `'[0-9a-fA-F]+'[hH]`},
		{"Ident", `[a-zA-Z][a-zA-Z0-9_-]*`},
		{"Int", `0|[1-9]\d*`},
		{"Punct", `\.\.|[!-/:-@\[\\` + "`" + `{-\~]`}, // Punctuation
	})

	compressSpace = regexp.MustCompile(`(?:\r?\n *)+`)
	smiParser     = participle.MustBuild[Module](
		participle.Lexer(smiLexer),       // Use the new Simple lexer
		participle.Unquote("ExtUTCTime"), // Use standard unquoting only for dates
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			token.Value = "OBJECT IDENTIFIER" // Ensure the mapped value is correct
			return token, nil
		}, "ObjectIdentifier"),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			token.Value = "OCTET STRING" // Ensure the mapped value is correct
			return token, nil
		}, "OctetString"),
		participle.Map(func(token lexer.Token) (lexer.Token, error) {
			// Manually unquote: remove outer quotes and handle basic escapes (\", \\).
			// This avoids issues with strconv.Unquote and raw newlines in multi-line strings.
			if len(token.Value) < 2 || token.Value[0] != '"' || token.Value[len(token.Value)-1] != '"' {
				// Should not happen based on the lexer rule, but check defensively.
				return token, fmt.Errorf("unexpected format for Text token: %q", token.Value)
			}
			// Slice off outer quotes
			content := token.Value[1 : len(token.Value)-1]

			// Handle basic escapes
			content = strings.ReplaceAll(content, `\\`, `\`)
			content = strings.ReplaceAll(content, `\"`, `"`)

			token.Value = content
			return token, nil
		}, "Text"),
		//participle.UseLookahead(2), // Keep commented out as original
		participle.Upper("ExtUTCTime", "BinString", "HexString"), // Keep Upper
		participle.Elide("Whitespace", "Comment"),                // Keep Elide
	)
)

// Parse function needs filename argument for v2
func Parse(filename string, r io.Reader) (*Module, error) {
	// Update Parse call signature - Parse now returns the struct and error
	return smiParser.Parse(filename, r)
}

// ParseFile already has filename, update Parse call inside
func ParseFile(path string) (*Module, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Open file: %w", err)
	}
	defer r.Close()
	// Pass filename to Parse
	module, err := Parse(path, r)
	if err != nil {
		// Add filename to error context if helpful
		return module, fmt.Errorf("Parse file %q: %w", path, err)
	}
	return module, nil
}
