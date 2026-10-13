#!/usr/bin/env python3
"""Print the URL a NIAC daemon answers on, from `ss -ltnH` output on stdin.

The daemon binds 127.0.0.1 by default, but an operator can bind it to one
address instead (CT304 runs `--listen 10.44.40.22:8445`), and then nothing
answers on loopback (#2613). The kernel's listener table is the one account of
where it bound that covers --listen, NIAC_LISTEN_ADDR and any unit drop-in.

Usage: ss -ltnH | listen-url.py PORT
Exits 1, printing nothing, when nothing listens on PORT yet.
"""

import sys

# A wildcard bind answers on loopback too, and loopback is the address a
# closed host firewall cannot mistake for a broken deployment.
WILDCARD_TO_LOOPBACK = {"0.0.0.0": "127.0.0.1", "*": "127.0.0.1", "[::]": "[::1]"}
LOOPBACK = {"127.0.0.1", "[::1]"}


def listen_url(ss_output: str, port: int) -> str | None:
    hosts = []
    for line in ss_output.splitlines():
        fields = line.split()
        if len(fields) < 4:
            continue
        host, _, local_port = fields[3].rpartition(":")
        if local_port != str(port):
            continue
        # ss appends the bound device as a zone ("127.0.0.53%lo").
        host = host[: host.index("]") + 1] if host.startswith("[") else host.split("%", 1)[0]
        hosts.append(WILDCARD_TO_LOOPBACK.get(host, host))
    if not hosts:
        return None
    host = next((h for h in hosts if h in LOOPBACK), hosts[0])
    return f"https://{host}:{port}"


def main() -> int:
    url = listen_url(sys.stdin.read(), int(sys.argv[1]))
    if url is None:
        return 1
    print(url)
    return 0


if __name__ == "__main__":
    sys.exit(main())
