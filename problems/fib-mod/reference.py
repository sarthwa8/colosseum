import sys

data = sys.stdin.read().split()
if len(data) != 2:
    raise ValueError("input violates constraints")
n, m = int(data[0]), int(data[1])
if not (0 <= n <= 10**18 and 1 <= m <= 10**9):
    raise ValueError("input violates constraints")


def fib_pair(k):
    """Fast doubling: returns (F(k) mod m, F(k+1) mod m) in O(log k)."""
    if k == 0:
        return (0, 1)
    a, b = fib_pair(k >> 1)
    c = (a * ((2 * b - a) % m)) % m  # F(2i)   = F(i) * (2*F(i+1) - F(i))
    d = (a * a + b * b) % m          # F(2i+1) = F(i)^2 + F(i+1)^2
    if k & 1:
        return (d, (c + d) % m)
    return (c, d)


print(fib_pair(n)[0] % m)
