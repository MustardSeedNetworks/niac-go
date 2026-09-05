#!/usr/bin/env python3
"""Self-test for check-fetch-error-states.py.

The gate is only worth having if it fails on the thing it exists to catch, so
CI runs this before the gate itself — the same shape as the other ratchets.
"""

from __future__ import annotations

import importlib.util
import sys
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "gate", Path(__file__).with_name("check-fetch-error-states.py")
)
gate = importlib.util.module_from_spec(spec)
spec.loader.exec_module(gate)

CASES = [
    (
        "destructuring only data and loading is unsatisfied",
        "const { data, loading } = useApiResource(fetchThing, []);",
        1,
    ),
    (
        "destructuring error is satisfied",
        "const { data, error } = useApiResource(fetchThing, []);",
        0,
    ),
    (
        "renaming error is satisfied",
        "const { data, error: thingError } = useApiResource(fetchThing, []);",
        0,
    ),
    (
        "an errorToast is satisfied",
        "const { data } = useApiResource(fetchThing, [], { errorToast: true });",
        0,
    ),
    (
        "a multi-line call is still seen",
        """const {
    data: topology,
    loading: topologyLoading,
  } = useApiResource(() => fetchTopology(session), [sessionId], poll);""",
        1,
    ),
    (
        "a multi-line call taking error is satisfied",
        """const {
    data: topology,
    error: topologyError,
  } = useApiResource(() => fetchTopology(session), [sessionId], poll);""",
        0,
    ),
    (
        "two calls are counted separately",
        "const { data: a } = useApiResource(f, []);\nconst { data: b, error } = useApiResource(g, []);",
        1,
    ),
    (
        # errorCount is not the error field, so this site still cannot show a
        # failure -- the word boundary is what keeps a near-miss from passing.
        "a word merely containing error does not satisfy it",
        "const { data, errorCount } = useApiResource(fetchThing, []);",
        1,
    ),
]


def main() -> int:
    failures = 0
    for name, source, want in CASES:
        got = len(gate.unsatisfied(source))
        if got != want:
            print(f"FAIL {name}: {got} unsatisfied, want {want}")
            failures += 1

    if failures:
        return 1

    print(f"check-fetch-error-states self-test: {len(CASES)} cases pass")
    return 0


if __name__ == "__main__":
    sys.exit(main())
