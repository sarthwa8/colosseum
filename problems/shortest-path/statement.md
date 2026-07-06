# Shortest Path

Given a directed weighted graph, find the length of the shortest path from
node `s` to node `t`, or `-1` if `t` is unreachable from `s`.

## Input
- The first line contains four integers `n m s t`: the number of nodes, the
  number of edges, the start node, and the target node. Nodes are numbered
  `1..n`.
- Each of the next `m` lines contains three integers `u v w`: a directed edge
  from `u` to `v` with weight `w`.

## Output
A single line with the shortest distance from `s` to `t`, or `-1` if there is
no path. If `s = t` the answer is `0`.

## Example
Input:
```
4 5 1 4
1 2 5
1 3 2
3 2 1
2 4 4
3 4 8
```
Output:
```
7
```
The shortest path is `1 → 3 → 2 → 4` with length `2 + 1 + 4 = 7`.
