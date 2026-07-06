# Expression Evaluator

Evaluate an arithmetic expression containing integer literals, `+`, `-`, `*`,
`/`, parentheses, and unary minus.

Rules:
- `*` and `/` bind tighter than `+` and `-`; operators of equal precedence
  evaluate left to right (`8 - 3 - 2` is `3`, not `7`).
- **Division truncates toward zero**: `7 / 2` is `3`, `-7 / 2` is `-3`,
  `7 / -2` is `-3`.
- Unary minus may appear before a number or a parenthesized expression:
  `-(2 + 3) * -4` is `20`.
- Spaces may appear anywhere between tokens.

## Input
A single line containing the expression.

## Output
A single line with the value of the expression.

## Example
Input:
```
2 + 3 * -(4 - 1)
```
Output:
```
-7
```
