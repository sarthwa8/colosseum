import sys

data = sys.stdin.read().split()
n = int(data[0])
nums = list(map(int, data[1:1 + n]))

best = cur = nums[0]
for x in nums[1:]:
    cur = max(x, cur + x)
    best = max(best, cur)
print(best)
