# Third-party notice — device symbols

`deviceSymbols.tsx` contains SVG path data vendored from **Material Symbols**
(Outlined style, weight 400).

- Source: <https://github.com/google/material-design-icons>
- Licence: Apache License 2.0
- Retrieved: 2026-09-11

The symbols used, and the device type each serves:

| Device type | Material symbol |
| -------------- | ----------------- |
| `router` | `router` |
| `switch` | `lan` |
| `access_point` | `wifi_tethering` |
| `firewall` | `security` |
| `server` | `dns` |
| `host` | `computer` |
| `workstation` | `desktop_windows` |
| `iot` | `sensors` |
| `layer3-switch` | `account_tree` |
| `printer` | `print` |
| `voip-phone` | `phone_in_talk` |
| `unknown` | `help` |

The paths are unmodified; only the surrounding `<svg>` element is ours, so that
the glyphs inherit `currentColor` and carry no accessible name of their own
(the node that contains them is labelled instead).

Apache-2.0 requires the licence text and attribution to travel with the work.
This file is that attribution; the licence text is at
<https://www.apache.org/licenses/LICENSE-2.0>.
