# Next Greater Element

For each element of an array, find the first element to its right that is
**strictly greater** than it. If no such element exists, the answer for that
position is `-1`.

## Input
- The first line contains one integer `n`, the number of elements.
- The second line contains `n` space-separated integers.

## Output
A single line with `n` space-separated integers: the answer for each position,
in order.

## Example
Input:
```
5
2 7 3 3 1
```
Output:
```
7 -1 -1 -1 -1
```
`2` is followed by `7`. Nothing to the right of `7` is greater. The two `3`s
have no *strictly* greater element after them (`3` is not greater than `3`),
and `1` is the last element, so its answer is `-1`.
