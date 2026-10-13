#!/usr/bin/env python3
"""deploy-validate probes the address the daemon bound, not always loopback."""

import importlib.util
from pathlib import Path
import unittest

SPEC = importlib.util.spec_from_file_location(
    "listen_url", Path(__file__).resolve().parent / "lab" / "listen-url.py"
)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


def ss(*locals_: str) -> str:
    return "".join(f"LISTEN 0      4096   {local} 0.0.0.0:*\n" for local in locals_)


class ListenURLTest(unittest.TestCase):
    def test_resolves(self):
        cases = [
            ("default loopback bind", ss("127.0.0.1:8445"), "https://127.0.0.1:8445"),
            ("CT304 --listen on its LAN address (#2613)", ss("10.44.40.22:8445"), "https://10.44.40.22:8445"),
            ("IPv4 wildcard", ss("0.0.0.0:8445"), "https://127.0.0.1:8445"),
            ("dual-stack wildcard", ss("*:8445"), "https://127.0.0.1:8445"),
            ("IPv6 wildcard", ss("[::]:8445"), "https://[::1]:8445"),
            ("IPv6 address", ss("[2001:db8::22]:8445"), "https://[2001:db8::22]:8445"),
            ("zone suffix", ss("10.44.40.22%eth0:8445"), "https://10.44.40.22:8445"),
            ("loopback preferred", ss("10.44.40.22:8445", "127.0.0.1:8445"), "https://127.0.0.1:8445"),
            ("other ports ignored", ss("0.0.0.0:22", "127.0.0.1:84450", "10.44.40.22:8445"), "https://10.44.40.22:8445"),
            ("nothing listening yet", ss("0.0.0.0:22"), None),
            ("empty table", "", None),
        ]
        for name, output, want in cases:
            with self.subTest(name):
                self.assertEqual(MODULE.listen_url(output, 8445), want)


if __name__ == "__main__":
    unittest.main()
