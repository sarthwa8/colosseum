# 0/1 Knapsack

You have a knapsack with weight capacity `W` and `n` items; item `i` has
weight `w_i` and value `v_i`. Each item can be taken at most once. Maximize
the total value of the items taken, subject to the total weight being at most
`W`.

## Input
- The first line contains two integers `n W`.
- The second line contains `n` integers: the weights `w_1 .. w_n`.
- The third line contains `n` integers: the values `v_1 .. v_n`.

## Output
A single line with the maximum achievable total value.

## Example
Input:
```
4 8
3 4 5 9
30 50 60 100
```
Output:
```
90
```
The best choice is items 1 and 3: weight `3 + 5 = 8 <= 8`, value
`30 + 60 = 90`. Items 2 and 3 would be worth `110` but weigh `9 > 8`, and
item 4 alone (`100`) weighs `9 > 8`.
