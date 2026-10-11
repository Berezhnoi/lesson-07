#!/usr/bin/env python3
"""Turn `go test -json` output into points.

Usage:
  go test -list '^Test(Stage|Hidden)' ./internal/app/ > list.txt
  go test -race -count=1 -json ./... > test.json
  python3 scripts/score.py --json test.json --list list.txt [--no-hidden]
                           [--gofmt ok|fail] [--vet ok|fail] [--out result.json]

Tests are grouped by name prefix: TestStage1_... to TestStage5_... and TestHidden_...
The list file gives the expected set of tests, so a test that never ran (build failure,
crash, timeout) counts as failed instead of silently shrinking the denominator.
"""

import argparse
import json
import re
import sys

STAGES = {
    1: ("Stage 1: skeleton, create, get", 6),
    2: ("Stage 2: validation and errors", 8),
    3: ("Stage 3: update, delete, ids", 8),
    4: ("Stage 4: list, pagination, filter", 12),
    5: ("Stage 5: CORS and concurrency", 6),
}
HIDDEN = ("Hidden edge cases", 20)
LINT = {"gofmt": 2, "vet": 3}

NAME_RE = re.compile(r"^Test(Stage(\d)|Hidden)_\w+$")
MAX_OUTPUT_LINES = 12


def read_expected(path):
    names = set()
    if not path:
        return names
    try:
        with open(path, encoding="utf-8", errors="replace") as f:
            for line in f:
                line = line.strip()
                if NAME_RE.match(line):
                    names.add(line)
    except OSError:
        pass
    return names


def read_results(path):
    """Return ({test: pass|fail|skip}, {test: [output lines]}) for top-level tests."""
    results = {}
    output = {}
    try:
        f = open(path, encoding="utf-8", errors="replace")
    except OSError:
        return results, output
    with f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except ValueError:
                continue
            test = ev.get("Test")
            if not test:
                continue
            top = test.split("/")[0]
            action = ev.get("Action")
            if action == "output":
                output.setdefault(top, []).append(ev.get("Output", "").rstrip("\n"))
            elif action in ("pass", "fail", "skip") and test == top:
                results[top] = action
    return results, output


def group_of(name):
    m = NAME_RE.match(name)
    if not m:
        return None
    return "hidden" if m.group(1) == "Hidden" else int(m.group(2))


def status_ok(value):
    return str(value).lower() in ("ok", "success", "pass", "true")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--json", required=True, help="output of go test -json")
    ap.add_argument("--list", help="output of go test -list (expected tests)")
    ap.add_argument("--no-hidden", action="store_true", help="ignore TestHidden_* tests")
    ap.add_argument("--gofmt", default="ok")
    ap.add_argument("--vet", default="ok")
    ap.add_argument("--out", help="write the result as JSON to this file")
    ap.add_argument("--quiet", action="store_true", help="print one CSV line instead of a report")
    args = ap.parse_args()

    results, output = read_results(args.json)
    expected = read_expected(args.list)
    if not expected:
        expected = {n for n in results if NAME_RE.match(n)}

    groups = {n: (title, weight) for n, (title, weight) in STAGES.items()}
    if not args.no_hidden:
        groups["hidden"] = HIDDEN

    rows = []
    failed = []
    total = 0.0
    maximum = 0.0
    for key, (title, weight) in groups.items():
        names = sorted(n for n in expected if group_of(n) == key)
        passed = [n for n in names if results.get(n) == "pass"]
        points = weight * len(passed) / len(names) if names else 0.0
        failed.extend(n for n in names if n not in passed and key != "hidden")
        rows.append((title, len(passed), len(names), points, weight))
        total += points
        maximum += weight

    lint_rows = []
    for name, weight in LINT.items():
        ok = status_ok(getattr(args, name))
        lint_rows.append((name, ok, weight if ok else 0))
        total += weight if ok else 0
        maximum += weight

    total = round(total, 2)
    result = {
        "points": total,
        "max": maximum,
        "groups": [
            {"title": t, "passed": p, "total": n, "points": round(pts, 2), "max": w}
            for t, p, n, pts, w in rows
        ],
        "lint": {name: ok for name, ok, _ in lint_rows},
        "failed_tests": failed,
    }
    if args.out:
        with open(args.out, "w", encoding="utf-8") as f:
            json.dump(result, f, indent=2)

    if args.quiet:
        print("%s,%s,%s" % (total, maximum, ";".join(failed)))
        return 0

    print("## Automatic check")
    print()
    print("| Part | Passed | Points |")
    print("|---|---|---|")
    for title, p, n, pts, w in rows:
        print("| %s | %d / %d | %.1f / %d |" % (title, p, n, pts, w))
    for name, ok, pts in lint_rows:
        print("| %s | %s | %d / %d |" % (name, "ok" if ok else "FAILED", pts, LINT[name]))
    print("| **Total** | | **%.1f / %d** |" % (total, maximum))

    if failed:
        print()
        print("### Failing tests")
        for name in failed:
            print()
            print("**%s**" % name)
            lines = [ln for ln in output.get(name, []) if ln.strip() and not ln.startswith("=== ")]
            lines = [ln for ln in lines if not ln.startswith("--- ")][:MAX_OUTPUT_LINES]
            if lines:
                print("```")
                for ln in lines:
                    print(ln)
                print("```")
            elif results.get(name) is None:
                print("did not run (compile error, crash or timeout)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
