# Input Constraints

- The first line contains two integers `n W` with `1 <= n <= 100` and
  `0 <= W <= 10^4`.
- The second line contains exactly `n` integer weights with
  `0 <= w_i <= 10^5` — a weight may be **zero**, and it may **exceed `W`**.
- The third line contains exactly `n` integer values with `0 <= v_i <= 10^9` —
  total values can exceed 32-bit range.
- `W = 0` is valid: only weight-zero items fit (and they always fit).

Any input satisfying these constraints is valid and must be handled correctly.
