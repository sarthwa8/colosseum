import sys

expr = sys.stdin.readline().rstrip("\n")
if len(expr) > 10**4 or any(c not in "0123456789+-*/() " for c in expr):
    raise ValueError("input violates constraints")
depth = 0
for c in expr:
    depth += (c == "(") - (c == ")")
    if depth > 100:
        raise ValueError("nesting too deep")
sys.setrecursionlimit(10000)

pos = 0


def peek():
    global pos
    while pos < len(expr) and expr[pos] == " ":
        pos += 1
    return expr[pos] if pos < len(expr) else ""


def parse_expr():
    global pos
    v = parse_term()
    while peek() in ("+", "-"):  # NOT `in "+-"`: peek() may return "" at EOF
        op = expr[pos]
        pos += 1
        w = parse_term()
        v = v + w if op == "+" else v - w
    return v


def parse_term():
    global pos
    v = parse_factor()
    while peek() in ("*", "/"):
        op = expr[pos]
        pos += 1
        w = parse_factor()
        if op == "*":
            v = v * w
        else:
            # Truncate toward zero: Python's // floors, which is wrong for
            # negatives (-7 // 2 == -4, we need -3).
            q = abs(v) // abs(w)
            v = -q if (v < 0) != (w < 0) else q
    return v


def parse_factor():
    global pos
    sign = 1
    while peek() == "-":  # iterative so `----5` can't blow the stack
        sign = -sign
        pos += 1
    c = peek()
    if c == "(":
        pos += 1
        v = parse_expr()
        if peek() != ")":
            raise ValueError("expected )")
        pos += 1
        return sign * v
    if not c.isdigit():
        raise ValueError("expected number")
    start = pos
    while pos < len(expr) and expr[pos].isdigit():
        pos += 1
    return sign * int(expr[start:pos])


result = parse_expr()
if peek() != "":
    raise ValueError("trailing input")
print(result)
