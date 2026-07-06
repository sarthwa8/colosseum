# Input Constraints

- The first line contains four integers `n m s t` with `1 <= n <= 10^4`,
  `0 <= m <= 10^5`, `1 <= s, t <= n`. `s` may equal `t`.
- Each of the next `m` lines contains three integers `u v w` with
  `1 <= u, v <= n` and `0 <= w <= 10^9`. Edges are directed. Self-loops
  (`u = v`), parallel edges, and zero-weight edges are all allowed.
- True shortest distances can be as large as `~10^14` — they do not fit in
  32 bits, and a hardcoded "infinity" of `10^9` is too small.

Any input satisfying these constraints is valid and must be handled correctly.
