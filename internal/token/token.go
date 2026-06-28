// Package token defines the lexical vocabulary of the goal language: a Kind for
// every lexeme and a first-class source Pos{Offset, Line, Col}.
//
// It is the foundation of the AST front-end (lexer → parser → ast). Unlike the
// splice passes, which re-lex the source and discard byte offsets every pass, the
// kinds and positions defined here are carried on the parsed tree so diagnostics,
// fmt, and the LSP can point at exact source locations.
//
// The goal-specific lexemes the splice approach faked are first-class here: postfix
// '?' (QUESTION), the fat arrow '=>' (FAT_ARROW, one token — not '=' then '>'), the
// ellipsis '...' (ELLIPSIS), and '///' doc-comment trivia (DOC_COMMENT). The
// contextual keywords implements/sealed/from/derive are deliberately NOT kinds: they
// lex as IDENT and the parser decides them positionally.
package token

// Kind enumerates every goal lexeme.
type Kind int

// Kind constants, grouped as in go/token. The unexported *_beg/*_end markers
// delimit the literal/operator/keyword ranges so they can be iterated.
const (
	ILLEGAL Kind = iota // unrecognized input
	EOF                 // end of input
	COMMENT             // // line or /* block */ comment
	DOC_COMMENT         // /// goal doc comment (retained as trivia)

	literalBeg
	IDENT  // main, x, parse
	INT    // 12345
	FLOAT  // 123.45
	IMAG   // 123.45i
	CHAR   // 'a'
	STRING // "abc"
	literalEnd

	operatorBeg
	// Arithmetic and bitwise operators.
	ADD     // +
	SUB     // -
	MUL     // *
	QUO     // /
	REM     // %
	AND     // &
	OR      // |
	XOR     // ^
	SHL     // <<
	SHR     // >>
	AND_NOT // &^

	// Assignment operators.
	ADD_ASSIGN     // +=
	SUB_ASSIGN     // -=
	MUL_ASSIGN     // *=
	QUO_ASSIGN     // /=
	REM_ASSIGN     // %=
	AND_ASSIGN     // &=
	OR_ASSIGN      // |=
	XOR_ASSIGN     // ^=
	SHL_ASSIGN     // <<=
	SHR_ASSIGN     // >>=
	AND_NOT_ASSIGN // &^=

	// Logical, comparison, and misc operators.
	LAND   // &&
	LOR    // ||
	ARROW  // <-
	INC    // ++
	DEC    // --
	EQL    // ==
	LSS    // <
	GTR    // >
	ASSIGN // =
	NOT    // !
	NEQ    // !=
	LEQ    // <=
	GEQ    // >=
	DEFINE // :=

	// goal-specific operators.
	QUESTION  // ?  (postfix unwrap)
	FAT_ARROW // => (match arm)
	ELLIPSIS  // ... (variadic / spread)

	// Delimiters.
	LPAREN    // (
	LBRACK    // [
	LBRACE    // {
	COMMA     // ,
	PERIOD    // .
	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }
	SEMICOLON // ;
	COLON     // :
	operatorEnd

	keywordBeg
	// Go reserved words.
	BREAK
	CASE
	CHAN
	CONST
	CONTINUE
	DEFAULT
	DEFER
	ELSE
	FALLTHROUGH
	FOR
	FUNC
	GO
	GOTO
	IF
	IMPORT
	INTERFACE
	MAP
	PACKAGE
	RANGE
	RETURN
	SELECT
	STRUCT
	SWITCH
	TYPE
	VAR
	// goal reserved words.
	MATCH
	ENUM
	ASSERT
	keywordEnd
)

// kindNames maps each Kind to its canonical spelling (for operators/keywords) or a
// descriptive name (for non-spellable kinds). Index by Kind.
var kindNames = [...]string{
	ILLEGAL:     "ILLEGAL",
	EOF:         "EOF",
	COMMENT:     "COMMENT",
	DOC_COMMENT: "DOC_COMMENT",

	IDENT:  "IDENT",
	INT:    "INT",
	FLOAT:  "FLOAT",
	IMAG:   "IMAG",
	CHAR:   "CHAR",
	STRING: "STRING",

	ADD:     "+",
	SUB:     "-",
	MUL:     "*",
	QUO:     "/",
	REM:     "%",
	AND:     "&",
	OR:      "|",
	XOR:     "^",
	SHL:     "<<",
	SHR:     ">>",
	AND_NOT: "&^",

	ADD_ASSIGN:     "+=",
	SUB_ASSIGN:     "-=",
	MUL_ASSIGN:     "*=",
	QUO_ASSIGN:     "/=",
	REM_ASSIGN:     "%=",
	AND_ASSIGN:     "&=",
	OR_ASSIGN:      "|=",
	XOR_ASSIGN:     "^=",
	SHL_ASSIGN:     "<<=",
	SHR_ASSIGN:     ">>=",
	AND_NOT_ASSIGN: "&^=",

	LAND:   "&&",
	LOR:    "||",
	ARROW:  "<-",
	INC:    "++",
	DEC:    "--",
	EQL:    "==",
	LSS:    "<",
	GTR:    ">",
	ASSIGN: "=",
	NOT:    "!",
	NEQ:    "!=",
	LEQ:    "<=",
	GEQ:    ">=",
	DEFINE: ":=",

	QUESTION:  "?",
	FAT_ARROW: "=>",
	ELLIPSIS:  "...",

	LPAREN:    "(",
	LBRACK:    "[",
	LBRACE:    "{",
	COMMA:     ",",
	PERIOD:    ".",
	RPAREN:    ")",
	RBRACK:    "]",
	RBRACE:    "}",
	SEMICOLON: ";",
	COLON:     ":",

	BREAK:       "break",
	CASE:        "case",
	CHAN:        "chan",
	CONST:       "const",
	CONTINUE:    "continue",
	DEFAULT:     "default",
	DEFER:       "defer",
	ELSE:        "else",
	FALLTHROUGH: "fallthrough",
	FOR:         "for",
	FUNC:        "func",
	GO:          "go",
	GOTO:        "goto",
	IF:          "if",
	IMPORT:      "import",
	INTERFACE:   "interface",
	MAP:         "map",
	PACKAGE:     "package",
	RANGE:       "range",
	RETURN:      "return",
	SELECT:      "select",
	STRUCT:      "struct",
	SWITCH:      "switch",
	TYPE:        "type",
	VAR:         "var",

	MATCH:  "match",
	ENUM:   "enum",
	ASSERT: "assert",
}

// String returns the canonical name of the kind: the source spelling for operators
// and keywords (e.g. "=>", "match") and a descriptive upper-case name for the
// non-spellable kinds (e.g. "IDENT", "EOF"). Out-of-range kinds render as "token(N)".
func (k Kind) String() string {
	if k >= 0 && int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return "token(" + itoa(int(k)) + ")"
}

// itoa is a tiny base-10 formatter so this package stays import-free.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// keywords maps the source spelling of each reserved word to its Kind. The
// contextual keywords implements/sealed/from/derive are intentionally absent: they
// lex as IDENT and the parser disambiguates them by position.
var keywords map[string]Kind

// operators maps the source spelling of each operator/delimiter to its Kind, so
// punctuation names round-trip through Lookup.
var operators map[string]Kind

func init() {
	keywords = make(map[string]Kind, keywordEnd-(keywordBeg+1))
	for k := keywordBeg + 1; k < keywordEnd; k++ {
		keywords[kindNames[k]] = k
	}
	operators = make(map[string]Kind, operatorEnd-(operatorBeg+1))
	for k := operatorBeg + 1; k < operatorEnd; k++ {
		operators[kindNames[k]] = k
	}
}

// Lookup returns the Kind for a reserved-word or operator/delimiter spelling and
// reports whether one was found. Unknown spellings — including ordinary identifiers
// and the contextual keywords — return (ILLEGAL, false). It is the inverse of String
// over the keyword and operator ranges.
func Lookup(name string) (Kind, bool) {
	if k, ok := keywords[name]; ok {
		return k, true
	}
	if k, ok := operators[name]; ok {
		return k, true
	}
	return ILLEGAL, false
}

// IsKeyword reports whether name is a goal reserved word. The contextual keywords
// (implements, sealed, from, derive) are not reserved and return false.
func IsKeyword(name string) bool {
	_, ok := keywords[name]
	return ok
}

// IsLiteral reports whether the kind is one of the literal classes (IDENT, INT, …).
func (k Kind) IsLiteral() bool { return literalBeg < k && k < literalEnd }

// IsOperator reports whether the kind is an operator or delimiter.
func (k Kind) IsOperator() bool { return operatorBeg < k && k < operatorEnd }

// IsKeyword reports whether the kind is a reserved word.
func (k Kind) IsKeyword() bool { return keywordBeg < k && k < keywordEnd }

// Pos is a source position: a 0-based byte Offset and a 1-based Line and Col. Offset
// is the canonical total order; Line/Col are carried for human-readable diagnostics.
type Pos struct {
	Offset int
	Line   int
	Col    int
}

// Less reports whether p precedes q in the source, ordered by byte Offset.
func (p Pos) Less(q Pos) bool { return p.Offset < q.Offset }

// IsValid reports whether p refers to a real source position. The zero Pos
// (Line 0) marks an absent position, e.g. an optional token that was not present.
func (p Pos) IsValid() bool { return p.Line > 0 }

// String renders the position as "line:col" for diagnostics.
func (p Pos) String() string { return itoa(p.Line) + ":" + itoa(p.Col) }

// Token is a single lexeme: its Kind, its source text (Lit, for identifiers,
// literals, and comments), and its starting Pos. The lexer emits these.
type Token struct {
	Kind Kind
	Lit  string
	Pos  Pos
}
