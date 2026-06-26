/**
 * tree-sitter grammar for the Goal language (.goal) — a thin dialect of Go.
 *
 * Covers Go's commonly-used surface as it appears in goal code plus goal's
 * additions: enum, match, sealed/implements, derive/from, Result/Option, the
 * postfix `?` unwrap, the `=>` match arrow, `...defaults`/`...derive` spreads,
 * and `///` doctest comments. Statement termination follows Go's automatic
 * semicolon insertion, handled by the external scanner.
 */

const PREC = {
  primary: 8,
  unary: 7,
  multiplicative: 6,
  additive: 5,
  comparative: 4,
  and: 3,
  or: 2,
  composite_literal: 1,
};

const multiplicative_ops = ["*", "/", "%", "<<", ">>", "&", "&^"];
const additive_ops = ["+", "-", "|", "^"];
const comparative_ops = ["==", "!=", "<", "<=", ">", ">="];

const terminator = ($) => $._automatic_semicolon;

function commaSep(rule) {
  return optional(commaSep1(rule));
}
function commaSep1(rule) {
  return seq(rule, repeat(seq(",", rule)), optional(","));
}

module.exports = grammar({
  name: "goal",

  externals: ($) => [$._automatic_semicolon],

  extras: ($) => [/\s/, $.comment, $.doc_comment],

  word: ($) => $.identifier,

  conflicts: ($) => [
    [$._expression, $.type_identifier],
    [$._expression, $.qualified_type],
    [$._expression, $.type_parameter],
    [$._expression, $.type_identifier, $.type_parameter],
    [$.type_parameter, $.type_identifier],
    [$._type, $.generic_type],
    [$.parameter, $.type_identifier],
    [$.field_declaration, $.type_identifier],
    [$.parameter, $.parenthesized_type],
    [$._const_spec],
    [$._const_spec, $.type_identifier],
    [$._var_spec],
    [$._var_spec, $.type_identifier],
    [$.function_declaration],
    [$.method_declaration],
    [$.function_type],
    [$.method_spec],
    [$.method_spec, $.type_identifier],
    [$.return_statement],
    [$.assert_statement],
    [$.assignment_statement],
    [$.short_var_declaration],
    [$.keyed_element, $._expression],
    [$.channel_type],
  ],

  rules: {
    source_file: ($) =>
      seq(
        optional(terminator($)),
        repeat(seq($._top_level_declaration, optional(terminator($)))),
      ),

    _top_level_declaration: ($) =>
      choice(
        $.package_clause,
        $.import_declaration,
        $.function_declaration,
        $.method_declaration,
        $.type_declaration,
        $.sealed_interface_declaration,
        $.enum_declaration,
        $.const_declaration,
        $.var_declaration,
        $.from_declaration,
        $.derive_declaration,
      ),

    // ---- package / import ----
    package_clause: ($) => seq("package", field("name", $.identifier)),

    import_declaration: ($) =>
      seq(
        "import",
        choice(
          $._import_spec,
          seq("(", optional(terminator($)), repeat(seq($._import_spec, optional(terminator($)))), ")"),
        ),
      ),
    _import_spec: ($) =>
      seq(optional(field("name", choice($.identifier, ".", "_"))), field("path", $._string_literal)),

    // ---- declarations ----
    function_declaration: ($) =>
      seq(
        "func",
        field("name", $.identifier),
        optional(field("type_parameters", $.type_parameters)),
        field("parameters", $.parameter_list),
        optional(field("result", $._result)),
        optional(field("body", $.block)),
      ),

    method_declaration: ($) =>
      seq(
        "func",
        field("receiver", $.parameter_list),
        field("name", $.identifier),
        field("parameters", $.parameter_list),
        optional(field("result", $._result)),
        optional(field("body", $.block)),
      ),

    from_declaration: ($) => seq("from", $.function_declaration),
    derive_declaration: ($) => seq("derive", $.function_declaration),

    type_parameters: ($) =>
      seq("[", commaSep1($.type_parameter), "]"),
    type_parameter: ($) =>
      seq(field("name", commaSep1($.identifier)), field("constraint", $._type)),

    parameter_list: ($) => seq("(", commaSep($.parameter), ")"),
    parameter: ($) =>
      choice(
        seq(field("name", commaSep1($.identifier)), optional("..."), field("type", $._type)),
        seq(optional("..."), field("type", $._type)),
      ),

    _result: ($) => choice($.parameter_list, $._type),

    type_declaration: ($) =>
      seq(
        "type",
        field("name", $.identifier),
        optional(field("type_parameters", $.type_parameters)),
        choice(seq("=", field("type", $._type)), field("type", $._type)),
      ),

    // goal: a sealed (closed) interface, named after the `interface` keyword.
    sealed_interface_declaration: ($) =>
      seq(
        "sealed",
        "interface",
        field("name", $.identifier),
        optional(field("type_parameters", $.type_parameters)),
        "{",
        optional(terminator($)),
        repeat(seq($._interface_member, optional(terminator($)))),
        "}",
      ),

    // goal: a struct's inline assertion that it satisfies one or more interfaces.
    implements_clause: ($) => seq("implements", commaSep1($._type)),

    const_declaration: ($) =>
      seq(
        "const",
        choice(
          $._const_spec,
          seq("(", optional(terminator($)), repeat(seq($._const_spec, optional(terminator($)))), ")"),
        ),
      ),
    _const_spec: ($) =>
      seq(
        field("name", commaSep1($.identifier)),
        optional(seq(optional(field("type", $._type)), "=", field("value", commaSep1($._expression)))),
      ),

    var_declaration: ($) =>
      seq(
        "var",
        choice(
          $._var_spec,
          seq("(", optional(terminator($)), repeat(seq($._var_spec, optional(terminator($)))), ")"),
        ),
      ),
    _var_spec: ($) =>
      seq(
        field("name", commaSep1($.identifier)),
        choice(
          seq(field("type", $._type), optional(seq("=", field("value", commaSep1($._expression))))),
          seq("=", field("value", commaSep1($._expression))),
        ),
      ),

    // ---- enums (goal) ----
    enum_declaration: ($) =>
      seq(
        "enum",
        field("name", $.identifier),
        "{",
        optional(terminator($)),
        repeat(seq($.enum_variant, optional(terminator($)))),
        "}",
      ),
    enum_variant: ($) =>
      seq(field("name", $.identifier), optional(field("payload", $.variant_payload))),
    variant_payload: ($) =>
      seq("{", commaSep1($.payload_field), "}"),
    // Canonical form is `name: Type`; the struct-style `name Type` is also accepted.
    payload_field: ($) =>
      seq(field("name", $.identifier), optional(":"), field("type", $._type)),

    // ---- types ----
    _type: ($) =>
      choice(
        $.type_identifier,
        $.qualified_type,
        $.generic_type,
        $.pointer_type,
        $.slice_type,
        $.array_type,
        $.map_type,
        $.channel_type,
        $.function_type,
        $.struct_type,
        $.interface_type,
        $.parenthesized_type,
      ),

    parenthesized_type: ($) => seq("(", $._type, ")"),
    type_identifier: ($) => alias($.identifier, $.type_identifier),
    qualified_type: ($) => seq(field("package", $.identifier), ".", field("name", $.identifier)),
    generic_type: ($) =>
      seq(field("name", choice($.type_identifier, $.qualified_type)), field("arguments", $.type_arguments)),
    type_arguments: ($) => seq("[", commaSep1($._type), "]"),
    pointer_type: ($) => prec(PREC.unary, seq("*", $._type)),
    slice_type: ($) => seq("[", "]", $._type),
    array_type: ($) => seq("[", field("length", $._expression), "]", $._type),
    map_type: ($) => seq("map", "[", field("key", $._type), "]", field("value", $._type)),
    channel_type: ($) =>
      prec.right(
        choice(seq("chan", $._type), seq("chan", "<-", $._type), seq("<-", "chan", $._type)),
      ),
    function_type: ($) => seq("func", $.parameter_list, optional($._result)),

    struct_type: ($) =>
      seq(
        "struct",
        optional(field("implements", $.implements_clause)),
        "{",
        optional(terminator($)),
        repeat(seq($.field_declaration, optional(terminator($)))),
        "}",
      ),
    field_declaration: ($) =>
      choice(
        seq(field("name", commaSep1($.identifier)), field("type", $._type), optional(field("tag", $._string_literal))),
        seq(optional("*"), field("type", choice($.type_identifier, $.qualified_type)), optional(field("tag", $._string_literal))),
      ),

    interface_type: ($) =>
      seq("interface", "{", optional(terminator($)), repeat(seq($._interface_member, optional(terminator($)))), "}"),
    _interface_member: ($) => choice($.method_spec, $._type),
    method_spec: ($) =>
      seq(field("name", $.identifier), field("parameters", $.parameter_list), optional(field("result", $._result))),

    // ---- statements ----
    block: ($) =>
      seq("{", optional(terminator($)), repeat(seq($._statement, optional(terminator($)))), "}"),

    _statement: ($) =>
      choice(
        $.block,
        $.return_statement,
        $.if_statement,
        $.for_statement,
        $.switch_statement,
        $.go_statement,
        $.defer_statement,
        $.fallthrough_statement,
        $.assert_statement,
        $.const_declaration,
        $.var_declaration,
        $.short_var_declaration,
        $.assignment_statement,
        $.inc_dec_statement,
        $.expression_statement,
      ),

    return_statement: ($) => seq("return", optional(commaSep1($._expression))),
    fallthrough_statement: ($) => "fallthrough",
    go_statement: ($) => seq("go", $._expression),
    defer_statement: ($) => seq("defer", $._expression),
    assert_statement: ($) =>
      seq("assert", field("condition", $._expression), optional(seq(",", commaSep1($._expression)))),

    if_statement: ($) =>
      seq(
        "if",
        optional(seq(field("initializer", $._simple_statement), ";")),
        field("condition", $._expression),
        field("consequence", $.block),
        optional(seq("else", field("alternative", choice($.block, $.if_statement)))),
      ),

    for_statement: ($) =>
      seq("for", optional(choice($._for_clause, $.range_clause, $._expression)), field("body", $.block)),
    _for_clause: ($) =>
      seq(
        optional(field("initializer", $._simple_statement)),
        ";",
        optional(field("condition", $._expression)),
        ";",
        optional(field("update", $._simple_statement)),
      ),
    range_clause: ($) =>
      seq(
        optional(seq(field("left", commaSep1($._expression)), choice(":=", "="))),
        "range",
        field("right", $._expression),
      ),

    // Go's switch is valid goal (match is preferred for enums, but the language is a
    // superset of Go). Covers expression and type switches; a `case` may list types.
    switch_statement: ($) =>
      seq(
        "switch",
        optional(seq(field("initializer", $._simple_statement), ";")),
        optional(field("value", $._simple_statement)),
        "{",
        repeat(choice($.case_clause, $.default_clause)),
        "}",
      ),
    case_clause: ($) =>
      seq(
        "case",
        commaSep1(choice($._expression, $._type)),
        ":",
        repeat(seq($._statement, optional(terminator($)))),
      ),
    default_clause: ($) =>
      seq("default", ":", repeat(seq($._statement, optional(terminator($))))),

    _simple_statement: ($) =>
      choice($.short_var_declaration, $.assignment_statement, $.inc_dec_statement, $.expression_statement),

    inc_dec_statement: ($) =>
      prec.left(seq(field("operand", $._expression), field("operator", choice("++", "--")))),

    short_var_declaration: ($) =>
      seq(field("left", commaSep1($._expression)), ":=", field("right", commaSep1($._expression))),

    assignment_statement: ($) =>
      seq(
        field("left", commaSep1($._expression)),
        field("operator", choice("=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", "&^=")),
        field("right", commaSep1($._expression)),
      ),

    expression_statement: ($) => $._expression,

    // ---- match (goal) ----
    match_expression: ($) =>
      seq("match", field("value", $._expression), "{", optional(terminator($)), repeat(seq($.match_arm, optional(terminator($)))), "}"),
    match_arm: ($) => seq(field("pattern", $._pattern), "=>", field("body", $._statement)),
    _pattern: ($) => choice($._expression, alias("_", $.rest_pattern)),

    // ---- expressions ----
    _expression: ($) =>
      choice(
        $.identifier,
        $.selector_expression,
        $.call_expression,
        $.index_expression,
        $.type_assertion,
        $.composite_literal,
        $.unary_expression,
        $.binary_expression,
        $.unwrap_expression,
        $.match_expression,
        $.parenthesized_expression,
        $._literal,
        $.true,
        $.false,
        $.nil,
        $.iota,
      ),

    parenthesized_expression: ($) => seq("(", $._expression, ")"),

    selector_expression: ($) =>
      prec(PREC.primary, seq(field("operand", $._expression), ".", field("field", $.identifier))),

    // The function may be a type for a conversion: `[]byte(p)`, `map[K]V(x)`, `(*T)(x)`.
    call_expression: ($) =>
      prec(
        PREC.primary,
        seq(
          field("function", choice($._expression, $.slice_type, $.array_type, $.map_type, $.pointer_type, $.parenthesized_type)),
          field("arguments", $.argument_list),
        ),
      ),
    // Arguments may be labeled (`Status.Active(since: now())`) — the goal variant/
    // construction call form — as well as positional or a trailing spread.
    argument_list: ($) =>
      seq("(", commaSep(choice($.labeled_argument, $.spread_element, $._expression)), ")"),
    labeled_argument: ($) =>
      seq(field("label", $.identifier), ":", field("value", $._expression)),

    index_expression: ($) =>
      prec(PREC.primary, seq(field("operand", $._expression), "[", commaSep1(choice($._expression, $._type)), "]")),

    // Type assertion `x.(T)`, and `x.(type)` in a type switch.
    type_assertion: ($) =>
      prec(PREC.primary, seq(field("operand", $._expression), ".", "(", choice($._type, "type"), ")")),

    composite_literal: ($) =>
      prec(
        PREC.composite_literal,
        seq(field("type", choice($.type_identifier, $.qualified_type, $.generic_type, $.slice_type, $.array_type, $.map_type)), field("body", $.literal_value)),
      ),
    literal_value: ($) =>
      seq("{", commaSep(choice($.keyed_element, $.spread_element, $._expression)), "}"),
    keyed_element: ($) =>
      seq(field("key", choice($.identifier, $._expression)), ":", field("value", choice($._expression, $.literal_value))),
    spread_element: ($) =>
      choice(seq("...", "defaults"), seq("...", "derive", optional($.argument_list)), seq("...", $._expression)),

    unary_expression: ($) =>
      prec(PREC.unary, seq(field("operator", choice("-", "+", "!", "^", "*", "&", "<-")), field("operand", $._expression))),

    binary_expression: ($) => {
      const table = [
        [PREC.multiplicative, choice(...multiplicative_ops)],
        [PREC.additive, choice(...additive_ops)],
        [PREC.comparative, choice(...comparative_ops)],
        [PREC.and, "&&"],
        [PREC.or, "||"],
      ];
      return choice(
        ...table.map(([precedence, operator]) =>
          prec.left(precedence, seq(field("left", $._expression), field("operator", operator), field("right", $._expression))),
        ),
      );
    },

    unwrap_expression: ($) => prec(PREC.primary, seq($._expression, "?")),

    // ---- literals ----
    _literal: ($) => choice($.int_literal, $.float_literal, $.imaginary_literal, $._string_literal, $.rune_literal),
    _string_literal: ($) => choice($.interpreted_string_literal, $.raw_string_literal),

    interpreted_string_literal: ($) =>
      seq('"', repeat(choice(token.immediate(prec(1, /[^"\\\n]+/)), $.escape_sequence)), '"'),
    raw_string_literal: ($) => seq("`", /[^`]*/, "`"),
    rune_literal: ($) => seq("'", choice(/[^'\\\n]/, $.escape_sequence), "'"),
    escape_sequence: ($) =>
      token.immediate(seq("\\", choice(/[abfnrtv\\'"]/, /x[0-9a-fA-F]{2}/, /u[0-9a-fA-F]{4}/, /U[0-9a-fA-F]{8}/, /[0-7]{3}/))),

    int_literal: ($) => token(choice(/0[xX][0-9a-fA-F_]+/, /0[oO]?[0-7_]+/, /0[bB][01_]+/, /[0-9][0-9_]*/)),
    float_literal: ($) => token(choice(/[0-9][0-9_]*\.[0-9_]*([eE][+-]?[0-9_]+)?/, /\.[0-9][0-9_]*([eE][+-]?[0-9_]+)?/, /[0-9][0-9_]*[eE][+-]?[0-9_]+/)),
    imaginary_literal: ($) => token(seq(/[0-9][0-9_]*(\.[0-9_]*)?([eE][+-]?[0-9_]+)?/, "i")),

    true: ($) => "true",
    false: ($) => "false",
    nil: ($) => "nil",
    iota: ($) => "iota",

    identifier: ($) => /[a-zA-Z_][a-zA-Z0-9_]*/,

    // ---- comments ----
    // `///` doc comments win over `//` line comments via higher token precedence.
    comment: ($) =>
      token(
        prec(-1, choice(seq("//", /.*/), seq("/*", /[^*]*\*+([^/*][^*]*\*+)*/, "/"))),
      ),
    doc_comment: ($) => token(prec(1, seq("///", /.*/))),
  },
});
