package lexer

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync" // Added for sync.Once
	"unicode"
	"unicode/utf8"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/lukeod/gosmi/parser/lexer/token" // Import our token package
	// "github.com/lukeod/gosmi/pkg/errors" // Assuming an error package exists or will be created
)

const eof = -1

// Lexer holds the state of the lexer and implements the participle lexer.Lexer interface.
type Lexer struct {
	input       string // the string being scanned
	filename    string // filename for position information
	start       int    // start position of this item
	pos         int    // current position in the input
	width       int    // width of last rune read from input
	line        int    // 1-based line number
	column      int    // 1-based column number in bytes
	startLine   int    // start line of the current token
	startColumn int    // start column of the current token

	// TODO: Add error collector if needed
	// collector *errors.DefaultErrorCollector
}

// NewLexer creates a new lexer for the given input string and filename.
func NewLexer(filename, input string) *Lexer {
	l := &Lexer{
		input:    input,
		filename: filename,
		line:     1,
		column:   1,
		// collector: errors.NewErrorCollector(), // Initialize error collector
	}
	return l
}

// --- Core Lexing Logic (implements lexer.Lexer) ---

// next returns the next rune in the input.
func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width

	// Update position tracking
	if r == '\n' {
		l.line++
		l.column = 1
	} else {
		// Handle tabs correctly if needed, otherwise just increment column
		l.column += w // Assuming simple byte column count for now
	}
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *Lexer) backup() {
	l.pos -= l.width
	// Correctly backup line and column
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	if r == '\n' {
		l.line--
		// Column needs to be recalculated based on the previous line's content
		// This is complex, for now, we might accept slight inaccuracy on backup
		// or avoid backup across newlines. A simpler approach is often sufficient.
		// Resetting to 0 or 1 is a common simplification if full accuracy isn't critical.
		l.column = 0 // Simplified: reset column (needs improvement for full accuracy)
	} else {
		l.column -= l.width
	}
}

// emit passes an item back to the client.
// Use this if using the channel approach for Participle.
// emitToken creates and returns a standard lexer.Token.
func (l *Lexer) emitToken(t token.TokenType) lexer.Token {
	tok := lexer.Token{
		Type:  lexer.TokenType(t), // Use our custom TokenType as the value
		Value: l.input[l.start:l.pos],
		Pos: lexer.Position{
			Filename: l.filename, // Use the stored filename
			Offset:   l.start,
			Line:     l.startLine,
			Column:   l.startColumn,
		},
	}
	l.start = l.pos // Move start for the next token
	l.startLine = l.line
	l.startColumn = l.column
	return tok
}

// ignore skips over the pending input before this point.
func (l *Lexer) ignore() {
	l.start = l.pos
	l.startLine = l.line
	l.startColumn = l.column
}

// acceptRun consumes a run of runes from the valid set.
func (l *Lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

// recordError adds an error to the collector (if implemented).
func (l *Lexer) recordError(message string) {
	// Placeholder: Implement error recording logic
	fmt.Printf("Lexer Error: Line %d, Col %d: %s\n", l.startLine, l.startColumn, message)
	// Example with collector:
	// pos := errors.Position{ Line: l.startLine, Column: l.startColumn }
	// l.collector.AddError(errors.NewErrorWithPosition(errors.LexerError, errors.SeverityError, pos, message))
}

// --- State Functions (Example Structure) ---
// type stateFn func(*Lexer) stateFn

// func lexText(l *Lexer) stateFn { ... } // Example state function

// run lexes the input by executing state functions until the state is nil.
// Use this if using the channel approach.
//
//	func (l *Lexer) run() {
//		for state := lexText; state != nil; { // Start state
//			state = state(l)
//		}
//	}

// Next returns the next token from the input, implementing the lexer.Lexer interface.
// Errors are handled by returning an ILLEGAL token. EOF is signaled by a token with Type lexer.EOF.
func (l *Lexer) Next() (lexer.Token, error) {
NextLoop: // Label for the outer loop
	for {
		// Set potential start position *before* skipping anything
		l.start = l.pos
		l.startLine = l.line
		l.startColumn = l.column

		r := l.peek() // Peek at the current character

		// Skip Whitespace
		if unicode.IsSpace(r) {
			l.skipWhitespace() // Consumes whitespace
			l.ignore()         // Update start pos *after* skipping
			continue NextLoop
		}

		// Skip Comments ('--')
		if r == '-' {
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '-' {
				// Consume the comment '--' and rest of line
				l.next()            // Consume first '-'
				l.next()            // Consume second '-'
				l.skipCommentRest() // Consume rest of line
				l.ignore()          // Update start pos *after* skipping
				continue NextLoop
			}
			// If not '--', fall through to token logic
		}

		// If we are here, the next character is not whitespace or the start of a comment.
		// The start position is already correctly set above.

		// Check for EOF *after* skipping
		if r == eof {
			// EOF token needs position at the end.
			// l.start is already correct (end of last token or start of input)
			// Participle expects EOF type to be lexer.EOF (-1)
			return l.emitToken(token.EOF), nil // Return EOF token, nil error
		}

		// Now consume the first character of the actual token
		r = l.next() // Consume the character we peeked at (r is updated)

		// --- Token Recognition Logic ---
		// Note: The start position (l.start, etc.) is already set correctly before this switch.
		switch {
		case r == ':':
			if l.peek() == ':' {
				l.next() // Consume ':'
				if l.peek() == '=' {
					l.next() // Consume '='
					return l.emitToken(token.Assign), nil
				}
				// Don't backup. Emit '::' as the illegal token.
				l.recordError("Expected '=' after '::'")
				// l.pos is already after '::', l.start is before the first ':'
				return l.emitToken(token.ILLEGAL), nil // Emits '::'
			}
			l.recordError("Illegal single ':'")
			return l.emitToken(token.ILLEGAL), nil
		case r == '.':
			if l.peek() == '.' {
				l.next() // Consume '.'
				return l.emitToken(token.Range), nil
			}
			return l.emitToken(token.Dot), nil
		case r == '|':
			return l.emitToken(token.Pipe), nil
		case r == '{':
			return l.emitToken(token.LBrace), nil
		case r == '}':
			return l.emitToken(token.RBrace), nil
		case r == '(':
			return l.emitToken(token.LPAREN), nil
		case r == ')':
			return l.emitToken(token.RPAREN), nil
		case r == ',':
			return l.emitToken(token.Comma), nil
		case r == ';':
			return l.emitToken(token.Semicolon), nil
		case r == '-':
			// We already handled '--' comments in the skipping logic above.
			// If we reach here with '-', it must be the Minus token.
			return l.emitToken(token.Minus), nil
		case r == '[':
			// Check for ASN1Tag specifically
			l.backup()                 // Backup the '['
			return l.lexASN1Tag(), nil // lexASN1Tag handles errors and returns ILLEGAL if needed
		case r == '"':
			l.backup() // Let lexText handle the '"'
			return l.lexText(), nil
		case r == '\'':
			l.backup() // Let lexQuotedString handle the '\''
			return l.lexQuotedString(), nil
		case unicode.IsDigit(r): // Check the consumed character
			l.backup() // Backup so lexNumber starts correctly
			return l.lexNumber(), nil
		case isIdentifierStart(r): // Check the consumed character
			l.backup()                    // Backup so lexIdentifier starts correctly
			return l.lexIdentifier(), nil // Lex the identifier first
		default:
			// The illegal character 'r' was already consumed by l.next() above
			l.recordError(fmt.Sprintf("Illegal character: %q", r))
			return l.emitToken(token.ILLEGAL), nil // Emits the single illegal char
		}
	}
}

// --- Helper Lexing Functions (returning lexer.Token) ---

// skipCommentRest consumes the rest of a comment line after '--' has been consumed.
func (l *Lexer) skipCommentRest() {
	for {
		r := l.next()
		if r == '\n' || r == eof {
			break
		}
	}
	// Don't backup newline, it's consumed
}

func (l *Lexer) lexIdentifier() lexer.Token {
	// Check for multi-word keywords first
	// Need to check the *exact* first word before calling peekAheadN

	// Check for OBJECT IDENTIFIER
	if l.peekAhead("OBJECT") {
		match, endPos := l.peekAheadN(len("OBJECT"), "IDENTIFIER")
		if match {
			// Consume the entire sequence including intermediate stuff
			// The value emitted will be l.input[l.start:endPos]
			// Need to manually advance pos and column correctly.
			// Store current pos before advancing.
			startPos := l.pos
			l.pos = endPos
			// Crude column update - doesn't account for newlines/tabs in comments
			l.column += (endPos - startPos)
			return l.emitToken(token.ObjectIdentifier)
		}
		// If peekAheadN failed, fall through to lex "OBJECT" as a regular Ident
	}

	// Check for OCTET STRING
	if l.peekAhead("OCTET") {
		match, endPos := l.peekAheadN(len("OCTET"), "STRING")
		if match {
			startPos := l.pos
			l.pos = endPos
			l.column += (endPos - startPos)
			return l.emitToken(token.OctetString)
		}
		// If peekAheadN failed, fall through to lex "OCTET" as a regular Ident
	}

	// Regular identifier
	l.next() // Consume the first character (already known to be identifier start)
	for {
		r := l.peek() // Peek at the next character

		// Check if it's a hyphen that might start a comment
		if r == '-' {
			// Peek ahead one more character
			if l.pos+l.width < len(l.input) { // Ensure there is a character after '-'
				nextR, _ := utf8.DecodeRuneInString(l.input[l.pos+l.width:])
				if nextR == '-' {
					// It's the start of a comment, stop the identifier *before* the first '-'
					break
				}
			}
			// If not '--', treat '-' as a valid identifier character below
		}

		if !isIdentifierChar(r) {
			// Not a letter, digit, or hyphen - end of identifier
			break
		}

		// It's a valid identifier character (letter, digit, or hyphen not starting a comment)
		l.next() // Consume identifier char
	}
	// Keywords are lexed as Ident, parser handles context
	return l.emitToken(token.Ident)
}

func (l *Lexer) lexNumber() lexer.Token {
	l.acceptRun("0123456789")
	// Could add checks for hex/binary/float here if needed
	return l.emitToken(token.Int)
}

func (l *Lexer) lexText() lexer.Token {
	startPosForCheck := l.start // Remember original start for ExtUTCTime check
	l.next()                    // Consume the opening '"'

	var builder strings.Builder
	// Estimate size: use current pos and original start pos. This is >= 1.
	builder.Grow(l.pos - startPosForCheck)

	atLineStart := true       // NEW: Flag to track if we are at the start of a line within the string
	lastCharWasSpace := false // NEW: Flag to track if the last written char was a space
	isTerminated := false

	for {
		r := l.next() // Consume next rune

		if r == '\\' {
			// Handle escape sequence
			escapeChar := l.next() // Consume the character *after* the backslash
			if escapeChar == eof {
				l.recordError("Unterminated escape sequence at end of string")
				break
			}
			// Write the *actual* escaped character (e.g., write '"' for '\"')
			builder.WriteRune(escapeChar)
			atLineStart = false      // Escaped char is not whitespace
			lastCharWasSpace = false // Escaped char is not space
			continue
		} else if r == '"' {
			isTerminated = true
			break // End of string
		} else if r == eof {
			l.recordError("Unterminated string literal")
			break
		}

		// Normalize CRLF and standalone CR to LF or space
		if r == '\r' {
			if l.peek() == '\n' {
				continue // Skip \r, handle \n in next iteration
			}
			r = ' ' // Treat standalone \r as space
		}

		isSpace := r == ' ' || r == '\t'
		isNewline := r == '\n'

		if isNewline {
			// Write the newline. Post-processing trims trailing whitespace anyway.
			builder.WriteRune('\n')
			atLineStart = true       // Next char will be at line start
			lastCharWasSpace = false // Newline is not a space
			continue
		} else if isSpace {
			if atLineStart {
				// Skip leading whitespace on a line
				continue
			}
			// Handle space compression: only write if the last written char wasn't a space
			if !lastCharWasSpace {
				builder.WriteRune(' ')
				lastCharWasSpace = true
			}
			// Don't update atLineStart, still processing spaces
			continue
		} else {
			// Regular character
			builder.WriteRune(r)
			atLineStart = false
			lastCharWasSpace = false
		}
	}

	// --- Post-loop processing ---

	if !isTerminated {
		// Emit the content read so far (including opening quote) as ILLEGAL
		// Reset pos back to start to emit the whole thing? No, emit what was consumed.
		l.start = startPosForCheck // Reset start to include opening quote for ILLEGAL token
		return l.emitToken(token.ILLEGAL)
	}

	// Check for ExtUTCTime using the *original* slice before emitting
	originalValueWithQuotes := l.input[startPosForCheck:l.pos]
	if len(originalValueWithQuotes) > 2 { // Need at least quotes + Z
		content := originalValueWithQuotes[1 : len(originalValueWithQuotes)-1] // Extract content without quotes
		if (len(content) == 11 || len(content) == 13) && (content[len(content)-1] == 'Z' || content[len(content)-1] == 'z') {
			isUTCTime := true
			for i := 0; i < len(content)-1; i++ { // Check only digits before Z
				if !unicode.IsDigit(rune(content[i])) {
					isUTCTime = false
					break
				}
			}
			if isUTCTime {
				// Emit ExtUTCTime with the *original* value (including quotes)
				l.start = startPosForCheck // Ensure start is correct for original value
				return l.emitToken(token.ExtUTCTime)
			}
		}
	}

	// If it's a terminated string and not ExtUTCTime, emit Text with the compressed value
	// Trim trailing newline/space from builder result
	finalContent := builder.String()
	for len(finalContent) > 0 {
		lastRune, size := utf8.DecodeLastRuneInString(finalContent)
		if lastRune == ' ' || lastRune == '\n' || lastRune == '\t' {
			finalContent = finalContent[:len(finalContent)-size]
		} else {
			break
		}
	}

	// Emit Text token with the processed value (no quotes)
	l.start = startPosForCheck + 1 // Set start *after* opening quote for the final value
	l.pos--                        // Set pos *before* closing quote for the final value
	// Temporarily modify input slice for emitToken (hacky but avoids changing emitToken)
	originalInput := l.input
	l.input = finalContent
	// originalStart := l.start // Removed unused variable
	originalPos := l.pos
	l.start = 0
	l.pos = len(finalContent)

	tok := l.emitToken(token.Text) // emitToken uses l.start:l.pos on l.input

	// Restore original lexer state
	l.input = originalInput
	l.start = startPosForCheck // Restore original start for next token
	l.pos = originalPos + 1    // Restore original pos (after closing quote) for next token
	l.startLine = l.line       // Ensure startLine/Col are updated after processing
	l.startColumn = l.column

	return tok
}

func (l *Lexer) lexQuotedString() lexer.Token {
	l.next() // Consume opening '\''

	contentStart := l.pos
	hasContentError := false // Restore declaration
	for {
		r := l.peek() // Peek
		if r == '\'' {
			l.next() // Consume closing '\''
			// isTerminated = true // No longer needed - Ensure this line is removed/remains commented
			break // End of quoted part
		} else if r == eof || r == '\n' { // Newlines not allowed in '...' strings
			l.recordError("Unterminated or multi-line single-quoted string")
			// Don't consume EOF/newline, let emitToken capture value up to current pos
			return l.emitToken(token.ILLEGAL) // Emit the unterminated part
		} else {
			// Consume the character inside the quotes
			l.next()
		}
	}
	contentEnd := l.pos - 1 // Position *before* the closing quote that was just consumed

	// Check suffix *after* consuming the closing quote
	suffix := l.peek()
	var tokType token.TokenType // Declare without initial value

	if suffix == 'H' || suffix == 'h' {
		l.next() // Consume the valid suffix
		tokType = token.HexString
		// Validate content *now*
		for i := contentStart; i < contentEnd; i++ {
			if !isHexDigit(rune(l.input[i])) {
				l.recordError(fmt.Sprintf("Invalid character '%c' in HexString", l.input[i]))
				hasContentError = true // Mark error
				break                  // Stop validation on first error
			}
		}
	} else if suffix == 'B' || suffix == 'b' {
		l.next() // Consume the valid suffix
		tokType = token.BinString
		// Validate content *now*
		for i := contentStart; i < contentEnd; i++ {
			if l.input[i] != '0' && l.input[i] != '1' {
				l.recordError(fmt.Sprintf("Invalid character '%c' in BinString", l.input[i]))
				hasContentError = true // Mark error
				break                  // Stop validation on first error
			}
		}
		// <<< End of BinString validation loop
	} else { // <<< Correctly placed else block
		// No valid suffix found ('H'/'B') or EOF reached after closing quote.
		l.recordError("Invalid or missing suffix for single-quoted string (expected 'H' or 'B')")
		// Check if there *is* a character immediately after the quote that isn't whitespace/EOF.
		// If so, consume it as part of the ILLEGAL token.
		if suffix != eof && !unicode.IsSpace(suffix) {
			l.next() // Consume the invalid suffix character
		}
		// Emit ILLEGAL token containing the single quotes, content, and potentially the invalid suffix.
		// l.pos is now after the closing quote and potentially after the consumed invalid suffix.
		// emitToken uses l.start:l.pos.
		tokType = token.ILLEGAL
		// No need to check hasContentError here, if suffix is wrong, it's ILLEGAL regardless of content.
	}

	// If content validation failed for H or B strings, override type to ILLEGAL
	if hasContentError {
		tokType = token.ILLEGAL
	}

	// emitToken uses l.start (before opening quote) and l.pos (after closing quote + suffix if valid/consumed)
	return l.emitToken(tokType)
}

func (l *Lexer) lexASN1Tag() lexer.Token {
	// Assumes starting '[' is already consumed by Next's caller via backup()
	l.next() // Consume '[' again

	startPos := l.pos // Remember start after '['

	// Skip whitespace after '['
	l.skipWhitespace()

	// Expect "APPLICATION"
	if !l.peekAhead("APPLICATION") {
		l.recordError("Expected 'APPLICATION' in ASN.1 Tag")
		l.pos = startPos // Reset pos to after '['
		l.backup()       // Backup '[' itself
		// Now emit '[' as ILLEGAL or maybe specific Bracket token?
		// Let's emit ILLEGAL for the '[' for now.
		l.next() // Consume '[' again
		return l.emitToken(token.ILLEGAL)
	}
	l.pos += len("APPLICATION")
	// Column update is tricky here, skip for now

	// Skip whitespace after APPLICATION
	l.skipWhitespace()

	// Expect digits
	digitStart := l.pos
	if !unicode.IsDigit(l.peek()) {
		l.recordError("Expected digits after 'APPLICATION' in ASN.1 Tag")
		// Emit the '[' + APPLICATION part as ILLEGAL?
		// Reset pos to start of tag and emit ILLEGAL up to current point
		l.pos = l.start    // Reset to '['
		l.next()           // Consume '['
		l.pos = digitStart // Advance to where digits were expected
		return l.emitToken(token.ILLEGAL)
	}
	l.acceptRun("0123456789")

	// Skip whitespace before ']'
	l.skipWhitespace()

	// Expect ']'
	if l.peek() != ']' {
		l.recordError("Expected ']' to close ASN.1 Tag")
		// Emit ILLEGAL up to the current position (which is likely EOF or the char after digits)
		// Do not reset pos, as it caused infinite loops.
		// l.pos = l.start // Reset to '[' - REMOVED
		// l.next()        // Consume '[' - REMOVED
		// // Find where ']' was expected - REMOVED Search Logic
		// endPos := l.pos
		// for endPos > l.start && l.input[endPos-1] != ']' {
		// 	endPos--
		// }
		// l.pos = endPos // Set pos to where ']' was expected - REMOVED
		// l.pos is currently at the character *after* the digits (or EOF)
		// l.start is at the beginning of the tag '['
		// Emit the whole "[APPLICATION 123" part as ILLEGAL
		return l.emitToken(token.ILLEGAL)
	}
	l.next() // Consume ']'

	return l.emitToken(token.ASN1Tag)
}

// --- Helper Lexing Functions (Continued) ---

// skipWhitespace consumes all contiguous whitespace characters.
func (l *Lexer) skipWhitespace() {
	for unicode.IsSpace(l.peek()) {
		l.next()
	}
}

// --- Character Predicates ---

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r)
}

func isIdentifierChar(r rune) bool {
	// Allow hyphen
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-'
}

func isHexDigit(r rune) bool {
	return ('0' <= r && r <= '9') || ('a' <= r && r <= 'f') || ('A' <= r && r <= 'F')
}

// --- Lookahead Helpers ---

// peekAhead checks if the input starting from the current position matches the given string.
func (l *Lexer) peekAhead(prefix string) bool {
	if l.pos+len(prefix) > len(l.input) {
		return false
	}
	return l.input[l.pos:l.pos+len(prefix)] == prefix
}

// peekAheadN checks if the input starting N bytes after the current position matches the given string,
// skipping intermediate whitespace AND comments.
// Returns true and the ending position of the match (after expected string) if successful.
func (l *Lexer) peekAheadN(offset int, expected string) (bool, int) {
	currentPos := l.pos + offset
	startLine := l.line // Track position for potential comment skipping
	startCol := l.column + offset

	// Skip intermediate whitespace and comments
	for currentPos < len(l.input) {
		r, w := utf8.DecodeRuneInString(l.input[currentPos:])
		if unicode.IsSpace(r) {
			currentPos += w
			if r == '\n' {
				startLine++
				startCol = 1
			} else {
				startCol += w
			}
			continue
		}
		// Check for comment start '--'
		// Ensure we don't go past end of input when checking second '-'
		if r == '-' && currentPos+w < len(l.input) && l.input[currentPos+w] == '-' {
			// Found comment, skip to end of line
			currentPos += w // Skip first '-'
			currentPos += w // Skip second '-' (assuming '-' is 1 byte width)

			// Skip until newline or EOF
			for currentPos < len(l.input) {
				rConsume, wConsume := utf8.DecodeRuneInString(l.input[currentPos:])
				currentPos += wConsume
				if rConsume == '\n' {
					startLine++
					startCol = 1
					break // Stop after consuming newline
				}
				if currentPos >= len(l.input) { // Check if we hit EOF
					break
				}
			}
			continue // Continue skipping after comment
		}
		// Found a non-space, non-comment character, stop skipping
		break
	}

	// Now check if the expected string matches at the adjusted position
	if currentPos+len(expected) > len(l.input) {
		return false, -1
	}
	match := l.input[currentPos:currentPos+len(expected)] == expected
	if match {
		// Return the position *after* the matched 'expected' string
		return true, currentPos + len(expected)
	}
	return false, -1
}

// --- Lexer Definition (implements lexer.Definition) ---

var (
	cachedSymbols map[string]lexer.TokenType
	symbolsOnce   sync.Once
)

// LexerDefinition implements the participle lexer.Definition interface.
type LexerDefinition struct{}

// Lex implements lexer.Definition.
func (d *LexerDefinition) Lex(filename string, r io.Reader) (lexer.Lexer, error) {
	inputBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input for lexing: %w", err)
	}
	return NewLexer(filename, string(inputBytes)), nil
}

// LexString implements lexer.Definition.
func (d *LexerDefinition) LexString(filename string, input string) (lexer.Lexer, error) {
	return NewLexer(filename, input), nil
}

// LexBytes implements lexer.Definition.
func (d *LexerDefinition) LexBytes(filename string, input []byte) (lexer.Lexer, error) {
	return NewLexer(filename, string(input)), nil
}

// Symbols implements lexer.Definition, caching the result.
func (d *LexerDefinition) Symbols() map[string]lexer.TokenType {
	symbolsOnce.Do(func() {
		cachedSymbols = make(map[string]lexer.TokenType)
		for tt, name := range token.Symbols {
			cachedSymbols[name] = lexer.TokenType(tt) // Use the custom TokenType value
		}
	})
	return cachedSymbols
}
