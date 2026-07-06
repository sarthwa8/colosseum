# Merge Intervals

Given `n` closed integer intervals `[l, r]`, merge every pair of intervals
that overlap or touch, and print the resulting disjoint intervals in
increasing order.

Two intervals overlap or touch when they share at least one integer point:
`[1, 2]` and `[2, 3]` merge into `[1, 3]`.

## Input
- The first line contains one integer `n`.
- Each of the next `n` lines contains two integers `l r` — one interval.
  The intervals are **not** necessarily sorted.

## Output
One line per merged interval, `l r`, in increasing order of `l`.

## Example
Input:
```
4
5 7
1 2
2 4
10 10
```
Output:
```
1 4
5 7
10 10
```
`[1,2]` and `[2,4]` touch at `2` and merge into `[1,4]`.
