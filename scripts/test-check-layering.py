#!/usr/bin/env python3
"""Self-tests for the layering gate."""

from __future__ import annotations

import subprocess
import tempfile
import unittest
from pathlib import Path

CHECKER = Path(__file__).with_name("check-layering.py")
MODULE = "github.com/MustardSeedNetworks/niac-go/"


def go_file(*imports: str, body: str = "") -> str:
    specs = "\n".join(f'\t"{MODULE}{path}"' for path in imports)
    return f"package p\n\nimport (\n\t\"fmt\"\n{specs}\n)\n\n{body}func f() {{ fmt.Println() }}\n"


class LayeringTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.write("internal/protocols/stack.go", go_file("internal/config"))
        self.write("internal/api/server.go", go_file("internal/protocols", "internal/daemon"))
        self.write("internal/daemon/daemon.go", go_file("internal/api"))

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def write(self, path: str, content: str) -> None:
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")

    def run_checker(self) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            ["python3", str(CHECKER), "--root", str(self.root)],
            capture_output=True,
            text=True,
            check=False,
        )

    def assert_fails_on(self, file: str, target: str) -> None:
        result = self.run_checker()
        self.assertEqual(result.returncode, 1, result.stdout)
        self.assertIn(f"{file} imports {target}", result.stdout)

    def test_outer_layers_compose_freely(self) -> None:
        result = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_every_outer_layer_is_forbidden(self) -> None:
        for target in (
            "internal/api",
            "internal/api/tokenstore",
            "internal/daemon",
            "internal/cliclient",
            "internal/acceptance/harness",
            "internal/wiretest",
            "cmd/niac",
        ):
            with self.subTest(target=target):
                self.write("internal/replay/controller.go", go_file(target))
                self.assert_fails_on("internal/replay/controller.go", target)

    def test_test_files_are_checked(self) -> None:
        self.write("internal/scenario/pack_test.go", go_file("internal/api"))
        self.assert_fails_on("internal/scenario/pack_test.go", "internal/api")

    def test_platform_tagged_files_are_checked(self) -> None:
        self.write("internal/capture/source_darwin.go", "//go:build darwin\n\n" + go_file("internal/daemon"))
        self.assert_fails_on("internal/capture/source_darwin.go", "internal/daemon")

    def test_single_and_aliased_imports(self) -> None:
        self.write("internal/library/one.go", f'package p\n\nimport srv "{MODULE}internal/api"\n')
        self.assert_fails_on("internal/library/one.go", "internal/api")

    def test_prefix_is_a_whole_path_segment(self) -> None:
        self.write("internal/apitools/x.go", go_file("internal/config"))
        self.write("internal/protocols/x.go", go_file("internal/apitools", "internal/daemonset"))
        result = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_comments_and_later_strings_are_not_imports(self) -> None:
        body = f'// import "{MODULE}internal/api"\nvar s = `import "{MODULE}internal/api"`\n\n'
        self.write("internal/config/x.go", go_file("internal/fabric", body=body))
        self.write("internal/fabric/y.go", f'package p\n\n/*\nimport "{MODULE}internal/daemon"\n*/\n')
        result = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_testdata_is_skipped(self) -> None:
        self.write("internal/library/testdata/fixture.go", go_file("internal/api"))
        result = self.run_checker()
        self.assertEqual(result.returncode, 0, result.stdout)

    def test_baseline_admits_and_must_shrink(self) -> None:
        self.write("internal/replay/controller.go", go_file("internal/api"))
        self.write("scripts/layering-baseline.txt", "internal/replay/controller.go  internal/api  # #1\n")
        self.assertEqual(self.run_checker().returncode, 0)

        self.write("internal/replay/controller.go", go_file("internal/config"))
        result = self.run_checker()
        self.assertEqual(result.returncode, 1, result.stdout)
        self.assertIn("lists internal/replay/controller.go -> internal/api, which is gone", result.stdout)


if __name__ == "__main__":
    unittest.main()
