# Input Constraints

- The first line contains a single integer `n` with `1 <= n <= 2*10^4`.
- The second line contains exactly `n` integers separated by single spaces.
- Each element `x` satisfies `-10^9 <= x <= 10^9`.
- "Greater" means strictly greater: for equal neighbors the answer is not the
  neighbor itself.

Any input satisfying these constraints is valid and must be handled correctly,
including arrays that are entirely non-increasing (every answer is `-1`) and
arrays with many duplicates.
