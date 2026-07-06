# Fibonacci Modulo

Compute the `n`-th Fibonacci number modulo `m`, where `F(0) = 0`, `F(1) = 1`,
and `F(k) = F(k-1) + F(k-2)`.

`n` can be astronomically large — a solution that iterates `n` times will not
finish in time.

## Input
A single line with two integers `n m`.

## Output
A single line with `F(n) mod m`.

## Example
Input:
```
10 1000
```
Output:
```
55
```
`F(10) = 55`.
