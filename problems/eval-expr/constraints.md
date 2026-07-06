# Input Constraints

- The input is one line: a syntactically valid expression of at most `10^4`
  characters, containing only digits, `+`, `-`, `*`, `/`, `(`, `)`, and spaces.
- Parentheses are nested at most `100` levels deep.
- Integer literals are between `0` and `10^9` (negative values are written with
  unary minus).
- Every divisor that is actually evaluated is nonzero.
- Every intermediate value fits in a signed 64-bit integer
  (`-9.2*10^18 .. 9.2*10^18`).
- Division truncates toward zero (so `-7 / 2 = -3`, not `-4`).

Any input satisfying these constraints is valid and must be handled correctly.
