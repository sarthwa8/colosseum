import sys

s = sys.stdin.readline().rstrip("\n")
if len(s) > 2 * 10**5 or any(not (c.islower() or c.isdigit()) for c in s):
    raise ValueError("input violates constraints")

best = 0
last = {}  # char -> most recent index
lo = 0     # window start
for i, c in enumerate(s):
    if c in last and last[c] >= lo:
        lo = last[c] + 1
    last[c] = i
    best = max(best, i - lo + 1)
print(best)
