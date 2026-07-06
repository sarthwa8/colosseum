import sys

data = sys.stdin.read().split()
n, W = int(data[0]), int(data[1])
if not (1 <= n <= 100 and 0 <= W <= 10**4) or len(data) != 2 + 2 * n:
    raise ValueError("input violates constraints")
ws = [int(x) for x in data[2 : 2 + n]]
vs = [int(x) for x in data[2 + n : 2 + 2 * n]]
if any(not 0 <= w <= 10**5 for w in ws) or any(not 0 <= v <= 10**9 for v in vs):
    raise ValueError("item out of range")

dp = [0] * (W + 1)  # dp[c] = best value with capacity exactly <= c
for w, v in zip(ws, vs):
    if w > W:
        continue
    if w == 0:
        # Zero-weight items are free value at every capacity.
        dp = [x + v for x in dp]
        continue
    for c in range(W, w - 1, -1):
        cand = dp[c - w] + v
        if cand > dp[c]:
            dp[c] = cand
print(dp[W])
