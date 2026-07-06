import sys

data = sys.stdin.read().split()
n, m = int(data[0]), int(data[1])
if not (1 <= n <= 100 and 1 <= m <= 100) or len(data) != 2 + n * m:
    raise ValueError("input violates constraints")
vals = [int(x) for x in data[2:]]
if any(abs(x) > 10**9 for x in vals):
    raise ValueError("element out of range")
grid = [vals[i * m : (i + 1) * m] for i in range(n)]

out = []
top, bottom, left, right = 0, n - 1, 0, m - 1
while top <= bottom and left <= right:
    for c in range(left, right + 1):
        out.append(grid[top][c])
    top += 1
    for r in range(top, bottom + 1):
        out.append(grid[r][right])
    right -= 1
    if top <= bottom:
        for c in range(right, left - 1, -1):
            out.append(grid[bottom][c])
        bottom -= 1
    if left <= right:
        for r in range(bottom, top - 1, -1):
            out.append(grid[r][left])
        left += 1
print(" ".join(map(str, out)))
