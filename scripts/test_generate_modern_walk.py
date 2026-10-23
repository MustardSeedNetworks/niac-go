"""Behavior checks for generated interface and memory bindings."""

import unittest

import generate_modern_walk as generator


class GeneratedBindingsTest(unittest.TestCase):
    def test_ten_gigabit_interfaces_retain_speed_in_if_high_speed(self) -> None:
        lines = generator.generate_interface_mib({"ports": 1, "uplinks": ["uplink"]})
        bindings = dict(line.split(" = ", 1) for line in lines)
        self.assertEqual(bindings[".1.3.6.1.2.1.2.2.1.5.2"], "Gauge32: 4294967295")
        self.assertEqual(bindings[".1.3.6.1.2.1.31.1.1.1.15.2"], "Gauge32: 10000")
        self.assertEqual(bindings[".1.3.6.1.2.1.2.2.1.5.1"], "Gauge32: 1000000000")
        self.assertEqual(bindings[".1.3.6.1.2.1.31.1.1.1.15.1"], "Gauge32: 1000")

    def test_authored_memory_pool_is_coherent_and_representable(self) -> None:
        lines = generator.generate_cisco_enterprise_mib({"model": "synthetic"})
        bindings = dict(line.split(" = ", 1) for line in lines)
        used = bindings[".1.3.6.1.4.1.9.9.48.1.1.1.5.1"]
        free = bindings[".1.3.6.1.4.1.9.9.48.1.1.1.6.1"]
        self.assertEqual(used.split(": ")[0], "Gauge32")
        self.assertEqual(free.split(": ")[0], "Gauge32")
        used_bytes = int(used.split(": ")[1])
        free_bytes = int(free.split(": ")[1])
        self.assertGreater(used_bytes, 0)
        self.assertGreater(free_bytes, 0)
        self.assertLessEqual(used_bytes, 4294967295)
        self.assertLessEqual(free_bytes, 4294967295)
        self.assertEqual(used_bytes + free_bytes, 4 * 1024**3)


if __name__ == "__main__":
    unittest.main()
