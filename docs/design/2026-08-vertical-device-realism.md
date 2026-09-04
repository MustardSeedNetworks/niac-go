# Vertical device realism

**Status:** Accepted (2026-08-07)

Decides what the seven scenario packs contain, how closely each simulated
device imitates its real counterpart, and where that imitation stops. Gates
M4-3 and M4-4; extends the capability decisions in
[customer scenario authoring](2026-07-customer-scenario-authoring-plan.md).

## The bar

A scanner pointed at a simulated device should identify it the way it would
identify the real thing. An MRI should answer as an MRI, a PLC as a PLC. That
means the ports it opens, the banner it returns, and the SNMP identity it
reports — not the application behind them.

## Where imitation stops

The authoring plan retires MQTT, IPMI and Redfish as outside NIAC's
network-simulation boundary. Vertical application protocols sit in the same
class, so they are bounded the same way: **implement the identity and discovery
surface, not the application workflow.**

| Protocol | Implement | Do not implement |
| --- | --- | --- |
| EtherNet/IP | List Identity on 44818/2222 — vendor ID, device type, product code, serial, product name | CIP object model, I/O connections |
| Modbus/TCP | Read Holding/Input Registers on 502, correct exception codes | Register-map semantics |
| DICOM | Association negotiation and C-ECHO on 104/11112 | C-STORE, modality worklist, image transfer |
| HL7 | MLLP framing and ACK on 2575 | Message-type semantics, ADT logic |
| JetDirect | Raw accept on 9100, PJL status | PCL or PostScript rendering |
| ONVIF | WS-Discovery Probe/ProbeMatch, device information | Media profiles, PTZ, streaming |
| SIP | OPTIONS response with a plausible User-Agent | Registration, call setup, RTP |

ONVIF and SIP are not optional extras. WS-Discovery is _how_ a camera is found,
and SIP is what distinguishes a phone from a host with a phone vendor's OUI.
Without them those devices do not read as cameras or phones to any scanner.

A deep protocol probe — a real DICOM C-STORE, a CIP connection — will fail.
That is the accepted limit. Simulating clinical or plant applications is a
different product.

## SNMP identity

Every profile currently resolves through `synth.VendorGeneric`, so an MRI
reports a generic `sysObjectID` and the vendor and model strings exist only as
NIAC metadata. Each role gets a vendor-accurate `sysObjectID` and `sysDescr`,
because SNMP identity is what most monitoring tools key on before any port scan
happens.

Where the sanitized walk corpus already holds a real walk for a device, its
values are used. Nothing is invented for a device we cannot substantiate; those
keep a generic identity and are recorded as such.

## Device mix

Real facilities hold many of a few things and one of several others. A hospital
runs dozens of infusion pumps and a single MRI. The generator's even rotation
produces as many MRIs as pumps, which reads wrong at a glance.

Endpoints are therefore **weighted**: common devices repeat, signature devices
appear once or twice. Weights stay deterministic so the Link-Live comparator
still works from authored truth.

Device counts per pack are unchanged (hospital 75, warehouse 69, manufacturing
69, campus 155, retail 95, service-provider 87, enterprise-scale 531). More
device _types_, same device _count_.

## Common tier

Phones, printers, cameras, UPS and PDUs appear on nearly every real network.
Every pack carries them, weighted per vertical — more cameras in a warehouse,
more phones on a campus, more UPS in a hospital riser.

| Device | Vendor examples | Identity |
| --- | --- | --- |
| IP phone | Cisco 8800, Poly, Yealink | SIP, CDP/LLDP-MED, SNMP |
| MFP / printer | HP LaserJet, Xerox, Canon | JetDirect 9100, IPP, Printer MIB |
| IP camera | Axis, Hanwha, Avigilon | ONVIF/WS-Discovery, RTSP, SNMP |
| UPS | APC Network Management Card | SNMP (UPS MIB) |
| PDU | APC, Vertiv | SNMP |
| Conference room | Cisco Room Kit, Poly Studio | SIP, CDP |
| NAS | Synology, QNAP | SMB, SNMP |
| Badge reader | HID, Lenel | SNMP |

## Server and storage tier

Every pack already generates six service servers — DNS, DHCP, app, file, NMS and
perf. That is the infrastructure floor, not the workload. Real facilities run
the systems their vertical depends on, and those systems are usually the busiest
and most interesting nodes on a discovery map.

Servers hang off the spine-leaf server block. Shared across packs:

| Class | Examples | Identity |
| --- | --- | --- |
| Virtualisation host | VMware ESXi, Hyper-V, Nutanix | SNMP, HTTPS |
| Enterprise storage | NetApp, Dell PowerStore, Pure | SNMP, HTTPS |
| NAS | Synology, QNAP | SMB, SNMP, HTTPS |
| Backup | Veeam, Commvault | SNMP, HTTPS |
| Directory / auth | AD DC, RADIUS, NAC (ISE, ClearPass) | LDAP, RADIUS, SNMP |

Per-vertical workload systems:

**Hospital** — **PACS archive** and vendor-neutral archive, EMR/EHR (Epic,
Cerner), lab information system, pharmacy system, and a medical imaging storage
array. PACS matters twice over: it is the busiest node in a real hospital
network, and it is the **DICOM peer the MRI and other modalities talk to**, so
both ends of that conversation answer C-ECHO and the story is coherent rather
than a lone imaging device talking to nothing.

**Manufacturing** — SCADA historian, MES server, OPC UA server, engineering
workstation.

**Warehouse** — WMS server, label/print server.

**Retail** — back-office server, inventory and pricing server.

**Campus** — file, print and virtualisation hosts; deliberately ordinary.

**Service provider** — NMS, RADIUS/AAA, provisioning, DNS resolvers.

## Client tier

Shared across campus, retail, hospital and service-provider packs.

| Class | Models |
| --- | --- |
| Windows laptop | Dell Latitude, Lenovo ThinkPad, HP EliteBook |
| Windows desktop | Dell OptiPlex, Lenovo ThinkCentre, HP EliteDesk |
| Thin client | Dell Wyse, HP t-series — heavily used in healthcare |
| Mac | MacBook Pro, iMac, Mac mini |
| Mobile | iPad, iPhone, Android handset |
| Chromebook | campus and education |

## Vertical device tiers

Signature devices per pack, on top of the common and client tiers.

**Hospital** — infusion pump (Baxter Sigma Spectrum), MRI (Siemens MAGNETOM
Vida), CT, ultrasound, X-ray, patient monitors (Philips MX850, GE B850),
ventilator, lab analyser, medication cabinet (Omnicell, Pyxis), nurse call
(Rauland, Ascom), pharmacy temperature sensor, pneumatic tube controller.

**Manufacturing** — PLC (Rockwell ControlLogix 5580), HMI (PanelView 5510),
robot controller (FANUC R-30iB Plus), VFD, machine-vision camera (Cognex),
industrial PC, RFID reader, weigh scale, safety controller, SCADA historian.

**Warehouse** — rugged handheld (Zebra TC58), label printer (Zebra ZT411),
forklift-mounted terminal, conveyor and sortation controller, RFID portal,
voice-picking base, AGV/AMR, print-and-apply.

**Retail** — POS (HP Engage One Pro), receipt printer, digital signage,
self-checkout, PIN pad, cash recycler, electronic shelf-label gateway,
back-office server.

**Campus** — client tier and common tier, deliberately plain. This pack shows
scale and clean structure rather than exotic devices.

**Service provider** — NOC workstation, test head, OLT, ONT, CPE router. The
current pack holds only a NOC workstation, which is the thinnest of the seven.

## Topology shape

### The spine is invariant, and modern

Every pack carries a full layered hierarchy, without exception. The _layers_ are
invariant; how they are realised follows current practice rather than the
textbook 2005 three-tier diagram.

```text
SD-WAN edge (HA) → NGFW cluster (HA) → core → distribution → access
                                                      ↳ Wi-Fi 6E/7 APs
                        ↳ spine/leaf server block
```

What "modern" means concretely:

| Aspect | Dated | What we build |
| --- | --- | --- |
| Edge | Two WAN routers, static | **SD-WAN edge pair**, dual transport |
| Firewall | Single inline | **NGFW HA cluster** |
| Core/dist | Always separate tiers | **Collapsed core** at small sites; separate tiers only where site size earns it |
| Server block | Flat server switch | **Spine-leaf** |
| Access | Large L2 domains, VLANs spanning | **Routed access** — L3 to the access edge, VLANs local |
| Access uplinks | 1G | **Multigig** to APs, 25/100G uplinks |
| Redundancy | STP-blocked links | **MLAG/vPC**, stacked or virtual-chassis access |
| Wireless | Hardware WLC in a closet | **Cloud-managed or virtual controller** |
| Wi-Fi | Wi-Fi 5/6 | **Wi-Fi 6E / 7** |

Differentiating a vertical never means dropping a layer. A warehouse still has a
core; a manufacturing plant still has firewalls. A map missing a tier reads as a
broken network rather than a different one. But a small retail store legitimately
runs a **collapsed core** rather than four distinct tiers — that is modern
practice, not a missing layer, and the map should show it that way.

**Every pack gets access points and wireless controllers.** Service-provider
currently generates none — 4 access switches per site and no AP or WLC at all —
which makes it both the thinnest pack and the one that breaks this rule. It gets
the wireless tier like the rest.

### What differs

Per-vertical shape is expressed in how the access layer is _organised_ and what
hangs beneath it, not by removing structure above it:

| Pack | Access-layer organisation |
| --- | --- |
| Hospital | building and floor; wards under floor distribution |
| Manufacturing | production cell and line; a cell switch per line |
| Warehouse | zone and dock; racking aisles and dock doors |
| Retail | store floor and back office, with a lane tier for checkout |
| Campus | building and floor, wide and shallow |
| Service provider | POP and access ring, with subscriber-facing aggregation |

Two maps from different verticals must not look alike. That is M4-3's stated
acceptance criterion — met by differing structure below the spine, with the
spine itself present and correct in all seven.

## Relationship to scenario authoring

The packs are the scenario library described in the authoring plan, so the
generator is the authoring backend rather than a parallel path. Profiles,
weights and per-vertical shapes are authoring primitives, which is why this
decision precedes M4-3 — verticals built outside the authoring model would be
rebuilt as templates later.
