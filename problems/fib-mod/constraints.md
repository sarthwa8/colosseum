# Input Constraints

- The input is one line with two integers `n m`.
- `0 <= n <= 10^18` — far too large to iterate; an `O(log n)` method
  (fast doubling or matrix exponentiation) is required.
- `1 <= m <= 10^9`. Note `m = 1` is valid (every answer is `0`).
- `F(0) = 0` and `F(1) = 1`.

Any input satisfying these constraints is valid and must be handled correctly,
including `n = 0`, `n = 1`, and `m = 1`.
