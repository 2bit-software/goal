#include "tree_sitter/parser.h"

// External scanner for goal's automatic statement termination (Go's automatic
// semicolon insertion). goal source carries no explicit `;`, so a statement ends
// at a newline (when the parser is at a point where a terminator is valid), or
// just before a closing `}`, or at end of input.

enum TokenType {
  AUTOMATIC_SEMICOLON,
};

void *tree_sitter_goal_external_scanner_create() { return NULL; }
void tree_sitter_goal_external_scanner_destroy(void *payload) {}
unsigned tree_sitter_goal_external_scanner_serialize(void *payload, char *buffer) { return 0; }
void tree_sitter_goal_external_scanner_deserialize(void *payload, const char *buffer, unsigned length) {}

static void skip(TSLexer *lexer) { lexer->advance(lexer, true); }

bool tree_sitter_goal_external_scanner_scan(void *payload, TSLexer *lexer, const bool *valid_symbols) {
  if (!valid_symbols[AUTOMATIC_SEMICOLON]) {
    return false;
  }

  lexer->result_symbol = AUTOMATIC_SEMICOLON;

  // Skip spaces/tabs/carriage returns on the current line. A newline, a closing
  // brace, or end of input ends the statement; any other token continues it.
  for (;;) {
    if (lexer->lookahead == 0) {
      return true; // end of input
    }
    if (lexer->lookahead == '}') {
      return true; // terminator inferred before a closing brace
    }
    if (lexer->lookahead == '\n') {
      skip(lexer); // consume the newline as part of the (zero-width) terminator
      return true;
    }
    if (lexer->lookahead == ' ' || lexer->lookahead == '\t' || lexer->lookahead == '\r') {
      skip(lexer);
      continue;
    }
    return false; // a real token follows on the same line; no terminator here
  }
}
