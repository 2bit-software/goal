# Business Spec — US-035

As a compiler engineer, I need `?` propagation lowered with proper gensym so the
magic `__goal_` prefix is retired.

## Acceptance Criteria
- `?` propagation is lowered using scope-aware generated identifiers, not the
  literal `__goal_` prefix.
- The 05-question-prop cases pass the behavioral tier and a test asserts the
  generated Go contains no `__goal_` substring.
