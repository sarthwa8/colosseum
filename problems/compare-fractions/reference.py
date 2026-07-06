import sys

data = sys.stdin.read().split()
t = int(data[0])
if not (1 <= t <= 10**5) or len(data) != 1 + 4 * t:
    raise ValueError("input violates constraints")

LIM = 10**18
out = []
for i in range(t):
    a, b, c, d = (int(x) for x in data[1 + 4 * i : 5 + 4 * i])
    if b == 0 or d == 0 or any(abs(v) > LIM for v in (a, b, c, d)):
        raise ValueError("query out of range")
    # Normalize signs into the numerators, then cross-multiply exactly.
    if b < 0:
        a, b = -a, -b
    if d < 0:
        c, d = -c, -d
    lhs, rhs = a * d, c * b
    out.append("<" if lhs < rhs else ">" if lhs > rhs else "=")
print("\n".join(out))
