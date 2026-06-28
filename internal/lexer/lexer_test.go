package lexer

import (
	"testing"

	"goal/internal/token"
)

// TestTokenizeMultiLineSample is the US-012 acceptance test: it tokenizes a
// multi-line sample and asserts the Kind, literal, and full position
// (Offset, Line, Col) of every token, ending in EOF. The leading tab on the
// "return" line and the blank second line exercise line increment and column
// reset (column is a 1-based byte column).
func TestTokenizeMultiLineSample(t *testing.T) {
	src := "package main\n\nfunc add(a int) int {\n\treturn a + 1\n}\n"

	want := []token.Token{
		{Kind: token.PACKAGE, Lit: "package", Pos: token.Pos{Offset: 0, Line: 1, Col: 1}},
		{Kind: token.IDENT, Lit: "main", Pos: token.Pos{Offset: 8, Line: 1, Col: 9}},

		{Kind: token.FUNC, Lit: "func", Pos: token.Pos{Offset: 14, Line: 3, Col: 1}},
		{Kind: token.IDENT, Lit: "add", Pos: token.Pos{Offset: 19, Line: 3, Col: 6}},
		{Kind: token.LPAREN, Pos: token.Pos{Offset: 22, Line: 3, Col: 9}},
		{Kind: token.IDENT, Lit: "a", Pos: token.Pos{Offset: 23, Line: 3, Col: 10}},
		{Kind: token.IDENT, Lit: "int", Pos: token.Pos{Offset: 25, Line: 3, Col: 12}},
		{Kind: token.RPAREN, Pos: token.Pos{Offset: 28, Line: 3, Col: 15}},
		{Kind: token.IDENT, Lit: "int", Pos: token.Pos{Offset: 30, Line: 3, Col: 17}},
		{Kind: token.LBRACE, Pos: token.Pos{Offset: 34, Line: 3, Col: 21}},

		{Kind: token.RETURN, Lit: "return", Pos: token.Pos{Offset: 37, Line: 4, Col: 2}},
		{Kind: token.IDENT, Lit: "a", Pos: token.Pos{Offset: 44, Line: 4, Col: 9}},
		{Kind: token.ADD, Pos: token.Pos{Offset: 46, Line: 4, Col: 11}},
		{Kind: token.INT, Lit: "1", Pos: token.Pos{Offset: 48, Line: 4, Col: 13}},

		{Kind: token.RBRACE, Pos: token.Pos{Offset: 50, Line: 5, Col: 1}},

		{Kind: token.EOF, Pos: token.Pos{Offset: 52, Line: 6, Col: 1}},
	}

	got := Tokens(src)
	if len(got) != len(want) {
		t.Fatalf("token count = %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i, w := range want {
		g := got[i]
		if g.Kind != w.Kind || g.Lit != w.Lit || g.Pos != w.Pos {
			t.Errorf("token %d = {%s %q %v}, want {%s %q %v}",
				i, g.Kind, g.Lit, g.Pos, w.Kind, w.Lit, w.Pos)
		}
	}
	if last := got[len(got)-1]; last.Kind != token.EOF {
		t.Errorf("last token Kind = %s, want EOF", last.Kind)
	}
}

// TestKeywordVsIdent confirms reserved words become keyword Kinds while ordinary
// words — and the contextual keywords implements/sealed/from/derive — stay IDENT.
func TestKeywordVsIdent(t *testing.T) {
	cases := []struct {
		src  string
		want token.Kind
	}{
		{"func", token.FUNC},
		{"for", token.FOR},
		{"return", token.RETURN},
		{"match", token.MATCH},
		{"enum", token.ENUM},
		{"assert", token.ASSERT},
		{"implements", token.IDENT},
		{"sealed", token.IDENT},
		{"from", token.IDENT},
		{"derive", token.IDENT},
		{"foo", token.IDENT},
	}
	for _, c := range cases {
		got := Tokens(c.src)
		if len(got) != 2 { // word + EOF
			t.Fatalf("%q: got %d tokens, want 2", c.src, len(got))
		}
		if got[0].Kind != c.want {
			t.Errorf("%q: Kind = %s, want %s", c.src, got[0].Kind, c.want)
		}
		if got[0].Lit != c.src {
			t.Errorf("%q: Lit = %q, want %q", c.src, got[0].Lit, c.src)
		}
	}
}

// TestLiterals confirms each literal class is recognized and classified.
func TestLiterals(t *testing.T) {
	cases := []struct {
		src  string
		want token.Kind
	}{
		{"42", token.INT},
		{"0xFF", token.INT},
		{"0b1010", token.INT},
		{"0o17", token.INT},
		{"1_000", token.INT},
		{"3.14", token.FLOAT},
		{".5", token.FLOAT},
		{"1e9", token.FLOAT},
		{"2.5e-3", token.FLOAT},
		{"1i", token.IMAG},
		{"'a'", token.CHAR},
		{`'\n'`, token.CHAR},
		{`"hi"`, token.STRING},
		{`"a\"b"`, token.STRING},
		{"`raw`", token.STRING},
	}
	for _, c := range cases {
		got := Tokens(c.src)
		if len(got) != 2 {
			t.Fatalf("%q: got %d tokens, want 2: %v", c.src, len(got), got)
		}
		if got[0].Kind != c.want {
			t.Errorf("%q: Kind = %s, want %s", c.src, got[0].Kind, c.want)
		}
		if got[0].Lit != c.src {
			t.Errorf("%q: Lit = %q, want %q", c.src, got[0].Lit, c.src)
		}
	}
}

// TestLongestMatchOperators confirms multi-character operators are matched
// greedily into a single token, not split into shorter ones.
func TestLongestMatchOperators(t *testing.T) {
	cases := []struct {
		src  string
		want token.Kind
	}{
		{":=", token.DEFINE},
		{"==", token.EQL},
		{"!=", token.NEQ},
		{"<=", token.LEQ},
		{">=", token.GEQ},
		{"<<", token.SHL},
		{">>", token.SHR},
		{"<<=", token.SHL_ASSIGN},
		{">>=", token.SHR_ASSIGN},
		{"&^", token.AND_NOT},
		{"&^=", token.AND_NOT_ASSIGN},
		{"&&", token.LAND},
		{"||", token.LOR},
		{"<-", token.ARROW},
		{"++", token.INC},
		{"--", token.DEC},
		{"+=", token.ADD_ASSIGN},
		{"+", token.ADD},
		{"<", token.LSS},
		{"=", token.ASSIGN},
	}
	for _, c := range cases {
		got := Tokens(c.src)
		if len(got) != 2 { // operator + EOF
			t.Fatalf("%q: got %d tokens, want 2: %v", c.src, len(got), got)
		}
		if got[0].Kind != c.want {
			t.Errorf("%q: Kind = %s, want %s", c.src, got[0].Kind, c.want)
		}
		if got[0].Pos != (token.Pos{Offset: 0, Line: 1, Col: 1}) {
			t.Errorf("%q: Pos = %v, want 0/1:1", c.src, got[0].Pos)
		}
	}
}

// TestIllegal confirms an unrecognized byte yields an ILLEGAL token (carrying the
// offending text) without panicking, and that lexing continues to EOF.
func TestIllegal(t *testing.T) {
	got := Tokens("a # b")
	want := []token.Kind{token.IDENT, token.ILLEGAL, token.IDENT, token.EOF}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w {
			t.Errorf("token %d Kind = %s, want %s", i, got[i].Kind, w)
		}
	}
	if got[1].Lit != "#" {
		t.Errorf("ILLEGAL Lit = %q, want %q", got[1].Lit, "#")
	}
	if got[1].Pos != (token.Pos{Offset: 2, Line: 1, Col: 3}) {
		t.Errorf("ILLEGAL Pos = %v, want offset 2 line 1 col 3", got[1].Pos)
	}
}

// TestGoalLexemes is a US-013 acceptance test: the goal-specific operators each
// lex to exactly one token of the expected kind — '?' is QUESTION, '=>' is a
// single FAT_ARROW (not '=' then '>'), and '...' is a single ELLIPSIS (not three
// PERIODs).
func TestGoalLexemes(t *testing.T) {
	cases := []struct {
		src  string
		want token.Kind
	}{
		{"?", token.QUESTION},
		{"=>", token.FAT_ARROW},
		{"...", token.ELLIPSIS},
	}
	for _, c := range cases {
		got := Tokens(c.src)
		if len(got) != 2 { // lexeme + EOF
			t.Fatalf("%q: got %d tokens, want 2 (lexeme + EOF): %v", c.src, len(got), got)
		}
		if got[0].Kind != c.want {
			t.Errorf("%q: Kind = %s, want %s", c.src, got[0].Kind, c.want)
		}
		if got[0].Pos != (token.Pos{Offset: 0, Line: 1, Col: 1}) {
			t.Errorf("%q: Pos = %v, want 0/1:1", c.src, got[0].Pos)
		}
	}
}

// TestGoalLexemesInContext guards the longest-match boundaries: '=>' beside an
// '=' is FAT_ARROW (not EQL), and '...' is distinct from a trailing single '.'.
func TestGoalLexemesInContext(t *testing.T) {
	// a => b : IDENT FAT_ARROW IDENT EOF
	got := Tokens("a => b")
	want := []token.Kind{token.IDENT, token.FAT_ARROW, token.IDENT, token.EOF}
	if len(got) != len(want) {
		t.Fatalf("`a => b`: got %d tokens, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w {
			t.Errorf("`a => b` token %d Kind = %s, want %s", i, got[i].Kind, w)
		}
	}

	// xs... : IDENT ELLIPSIS EOF (ellipsis is one token, not three PERIODs)
	got = Tokens("xs...")
	want = []token.Kind{token.IDENT, token.ELLIPSIS, token.EOF}
	if len(got) != len(want) {
		t.Fatalf("`xs...`: got %d tokens, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w {
			t.Errorf("`xs...` token %d Kind = %s, want %s", i, got[i].Kind, w)
		}
	}

	// f(x)? : the postfix unwrap is a lone QUESTION after the call
	got = Tokens("f(x)?")
	want = []token.Kind{token.IDENT, token.LPAREN, token.IDENT, token.RPAREN, token.QUESTION, token.EOF}
	if len(got) != len(want) {
		t.Fatalf("`f(x)?`: got %d tokens, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w {
			t.Errorf("`f(x)?` token %d Kind = %s, want %s", i, got[i].Kind, w)
		}
	}
}

// TestDocCommentTrivia is a US-013 acceptance test: '///' and the rest of its
// line lex to a single DOC_COMMENT token whose text is retained as trivia,
// distinct from an ordinary '//' COMMENT.
func TestDocCommentTrivia(t *testing.T) {
	got := Tokens("/// doc line\nx // note")
	want := []struct {
		kind token.Kind
		lit  string
	}{
		{token.DOC_COMMENT, "/// doc line"},
		{token.IDENT, "x"},
		{token.COMMENT, "// note"},
		{token.EOF, ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w.kind || got[i].Lit != w.lit {
			t.Errorf("token %d = {%s %q}, want {%s %q}", i, got[i].Kind, got[i].Lit, w.kind, w.lit)
		}
	}
}

// TestComments confirms // line and /* */ block comments are recognized as
// COMMENT tokens, distinct from the /// DOC_COMMENT trivia (see
// TestDocCommentTrivia).
func TestComments(t *testing.T) {
	got := Tokens("x // hi\n/* a */ y")
	want := []struct {
		kind token.Kind
		lit  string
	}{
		{token.IDENT, "x"},
		{token.COMMENT, "// hi"},
		{token.COMMENT, "/* a */"},
		{token.IDENT, "y"},
		{token.EOF, ""},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Kind != w.kind || got[i].Lit != w.lit {
			t.Errorf("token %d = {%s %q}, want {%s %q}", i, got[i].Kind, got[i].Lit, w.kind, w.lit)
		}
	}
}
