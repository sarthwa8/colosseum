import sys

data = sys.stdin.read().split()
n = int(data[0])
if not (1 <= n <= 2 * 10**4) or len(data) != n + 1:
    raise ValueError("input violates constraints")
nums = list(map(int, data[1:]))
if any(abs(x) > 10**9 for x in nums):
    raise ValueError("element out of range")

ans = [-1] * n
stack = []  # indices with undecided answers, values non-increasing
for i, x in enumerate(nums):
    while stack and nums[stack[-1]] < x:
        ans[stack.pop()] = x
    stack.append(i)
print(" ".join(map(str, ans)))
