# Compare Fractions

For each query, compare the fractions `a/b` and `c/d` exactly and print `<`,
`>`, or `=`.

Denominators may be **negative**: `1/-2` is the same number as `-1/2`.
Comparison must be exact — the values can be far too large for floating point.

## Input
- The first line contains one integer `T`, the number of queries.
- Each of the next `T` lines contains four integers `a b c d`.

## Output
For each query, one line containing `<` if `a/b < c/d`, `>` if `a/b > c/d`,
or `=` if they are equal.

## Example
Input:
```
3
1 2 2 4
1 -2 1 2
3 7 2 5
```
Output:
```
=
<
>
```
`1/2 = 2/4`; `1/-2 = -0.5 < 0.5`; `3/7 ≈ 0.4286 > 2/5 = 0.4`.
