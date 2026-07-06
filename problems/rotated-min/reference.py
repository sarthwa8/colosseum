import sys

data = sys.stdin.read().split()
n = int(data[0])
if not (1 <= n <= 2 * 10**5) or len(data) != n + 1:
    raise ValueError("input violates constraints")
nums = list(map(int, data[1:]))
if any(abs(x) > 10**9 for x in nums):
    raise ValueError("element out of range")

# Validate the rotated-sorted-distinct structure so out-of-spec attack inputs
# are rejected: at most one descent, and if one exists the array wraps around.
descents = sum(1 for i in range(n - 1) if nums[i] >= nums[i + 1])
if descents > 1 or (descents == 1 and nums[-1] >= nums[0]) or len(set(nums)) != n:
    raise ValueError("not a rotation of a sorted array of distinct elements")

print(min(nums))
