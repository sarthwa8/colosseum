import heapq
import sys

data = sys.stdin.buffer.read().split()
n, m, s, t = (int(x) for x in data[:4])
if not (1 <= n <= 10**4 and 0 <= m <= 10**5 and 1 <= s <= n and 1 <= t <= n):
    raise ValueError("input violates constraints")
if len(data) != 4 + 3 * m:
    raise ValueError("wrong edge count")

adj = [[] for _ in range(n + 1)]
for i in range(m):
    u, v, w = (int(x) for x in data[4 + 3 * i : 7 + 3 * i])
    if not (1 <= u <= n and 1 <= v <= n and 0 <= w <= 10**9):
        raise ValueError("edge out of range")
    adj[u].append((v, w))

INF = float("inf")
dist = [INF] * (n + 1)
dist[s] = 0
pq = [(0, s)]
while pq:
    d, u = heapq.heappop(pq)
    if d > dist[u]:
        continue
    if u == t:
        break
    for v, w in adj[u]:
        nd = d + w
        if nd < dist[v]:
            dist[v] = nd
            heapq.heappush(pq, (nd, v))

print(-1 if dist[t] == INF else dist[t])
