// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * Static data for HelpDrawer sections.
 */

import type { Feature, GlossaryEntry, Shortcut } from './types';

export const FEATURES: Feature[] = [
  {
    title: 'Dashboard',
    description: 'Live counters, run snapshots, and alert state for the active NIAC stack.',
    path: '/',
  },
  {
    title: 'Simulation',
    description: 'Pick a network, choose an interface, start or stop the daemon.',
    path: '/runtime',
  },
  {
    title: 'Running Devices',
    description: 'Read-only view of the devices the daemon is currently simulating.',
    path: '/devices',
  },
  {
    title: 'Devices',
    description: 'Reusable device definitions. Edit, clone, delete the library on disk.',
    path: '/device-config',
  },
  {
    title: 'Topology',
    description: 'Visual graph of the configured network plus the live neighbor table.',
    path: '/topology',
  },
  {
    title: 'Traffic',
    description: 'Inject controlled errors into the simulation and replay captured PCAPs.',
    path: '/traffic',
  },
  {
    title: 'Logs',
    description: 'Live log stream from the daemon, with per-protocol debug-level controls.',
    path: '/debug',
  },
  {
    title: 'Packets',
    description: 'Live packets crossing the simulation — hex view, BPF filter, save to PCAP.',
    path: '/packets',
  },
  {
    title: 'SNMP Walks',
    description: 'Validate and auto-fix SNMP walk files used by the simulated SNMP agents.',
    path: '/walk-validator',
  },
  {
    title: 'Compare & Merge',
    description: 'Compare two YAML network configs side-by-side and merge changes between them.',
    path: '/config-diff',
  },
];

export const GLOSSARY: GlossaryEntry[] = [
  {
    term: 'ARP',
    definition:
      'Address Resolution Protocol - Maps IP addresses to MAC addresses on a local network.',
    category: 'protocol',
  },
  {
    term: 'CDP',
    definition: 'Cisco Discovery Protocol - Proprietary protocol for discovering Cisco devices.',
    category: 'protocol',
  },
  {
    term: 'DHCP',
    definition:
      'Dynamic Host Configuration Protocol - Automatically assigns IP addresses to devices.',
    category: 'protocol',
  },
  {
    term: 'DNS',
    definition: 'Domain Name System - Translates domain names to IP addresses.',
    category: 'protocol',
  },
  {
    term: 'ICMP',
    definition: 'Internet Control Message Protocol - Used for network diagnostics (ping).',
    category: 'protocol',
  },
  {
    term: 'LLDP',
    definition: 'Link Layer Discovery Protocol - Vendor-neutral protocol for device discovery.',
    category: 'protocol',
  },
  {
    term: 'SNMP',
    definition: 'Simple Network Management Protocol - Monitors and manages network devices.',
    category: 'protocol',
  },
  {
    term: 'MAC Address',
    definition: 'Media Access Control address - Unique hardware identifier for network interfaces.',
    category: 'concept',
  },
  {
    term: 'PCAP',
    definition: 'Packet Capture - File format for storing captured network traffic.',
    category: 'concept',
  },
  {
    term: 'VLAN',
    definition: 'Virtual LAN - Partitions a physical network into multiple virtual networks.',
    category: 'concept',
  },
  {
    term: 'Router',
    definition: 'Network device that forwards packets between different networks.',
    category: 'device',
  },
  {
    term: 'Switch',
    definition: 'Network device that connects devices within the same network (Layer 2).',
    category: 'device',
  },
  {
    term: 'Firewall',
    definition: 'Security device that monitors and filters network traffic.',
    category: 'device',
  },
];

export const SHORTCUTS: Shortcut[] = [
  { keys: ['Esc'], description: 'Close drawer or modal', category: 'general' },
  { keys: ['?'], description: 'Open help', category: 'general' },
  { keys: ['/', ','], description: 'Open settings', category: 'general' },
  { keys: ['g', 'h'], description: 'Go to Command Center', category: 'navigation' },
  { keys: ['g', 'r'], description: 'Go to Runtime Control', category: 'navigation' },
  { keys: ['g', 'd'], description: 'Go to Devices', category: 'navigation' },
  { keys: ['g', 't'], description: 'Go to Topology', category: 'navigation' },
  { keys: ['Ctrl', 'k'], description: 'Quick search', category: 'actions' },
  { keys: ['Ctrl', 's'], description: 'Save current configuration', category: 'actions' },
  { keys: ['Ctrl', 'Enter'], description: 'Run/execute action', category: 'actions' },
];
