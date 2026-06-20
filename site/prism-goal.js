/* goal grammar for Prism — hand-written for the playground (not vendored).
   goal is Go plus a tiny closed delta, so we extend Prism's vendored Go grammar
   (prism.js) rather than redefine it: just the `enum`/`match` keywords and the
   `=>` match-arm arrow. Member access (Result.Ok), generic types (Option[T]) and
   variant fields (since:) already render correctly via the inherited Go rules.
   Used by the <code-input> editor and the static lang-goal doc snippets. */
Prism.languages.goal = Prism.languages.extend("go", {
  keyword:
    /\b(?:enum|match|break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go(?:to)?|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b/,
});
Prism.languages.insertBefore("goal", "operator", {
  arrow: { pattern: /=>/, alias: "operator" },
});
