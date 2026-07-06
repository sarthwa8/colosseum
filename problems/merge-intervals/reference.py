import sys

data = sys.stdin.read().split()
n = int(data[0])
if not (1 <= n <= 10**4) or len(data) != 1 + 2 * n:
    raise ValueError("input violates constraints")
ivs = []
for i in range(n):
    l, r = int(data[1 + 2 * i]), int(data[2 + 2 * i])
    if not (-10**9 <= l <= r <= 10**9):
        raise ValueError("interval out of range")
    ivs.append((l, r))

ivs.sort()
out = []
cl, cr = ivs[0]
for l, r in ivs[1:]:
    if l <= cr:  # overlapping or touching (closed intervals)
        cr = max(cr, r)
    else:
        out.append((cl, cr))
        cl, cr = l, r
out.append((cl, cr))
print("\n".join(f"{l} {r}" for l, r in out))
