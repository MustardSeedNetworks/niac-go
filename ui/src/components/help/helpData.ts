import type { ReactNode } from 'react';

export interface Feature {
  title: string;
  description: string;
  path: string;
  badge?: string;
}

export interface GlossaryEntry {
  term: string;
  definition: string;
  category: 'protocol' | 'concept' | 'device';
}

export interface Shortcut {
  keys: string[];
  description: string;
  category: 'navigation' | 'actions' | 'general';
}

export type HelpTab = 'features' | 'glossary' | 'shortcuts';

export interface TabConfig {
  id: HelpTab;
  label: string;
  icon: ReactNode;
}

export const FEATURES: Feature[] = [
  {
    title: 'Command Center',
    description: 'Live counters, run snapshots, and automation status for the active NIAC stack.',
    path: '/',
  },
  {
    title: 'Runtime Control',
    description: 'Monitor runtime status, view network interfaces, and manage NIAC configuration.',
    path: '/runtime',
  },
  {
    title: 'Devices & Config',
    description:
      'Review YAML-derived devices, SNMP walks, DHCP/DNS personas, and packet playback targets.',
    path: '/devices',
  },
  {
    title: 'Config Builder',
    description: 'Create, edit, and manage device configurations with a visual interface.',
    path: '/device-config',
    badge: 'New',
  },
  {
    title: 'Topology View',
    description: 'LLDP/CDP/EDP/FDP visibility for verifying intent before exporting to Graphviz.',
    path: '/topology',
  },
  {
    title: 'Analysis & Playback',
    description: 'Replay PCAPs, inspect SNMP walks, and publish bundles directly from the UI.',
    path: '/analysis',
  },
  {
    title: 'Traffic Injection',
    description: 'Configure and execute traffic injection scenarios with real-time feedback.',
    path: '/injection',
  },
  {
    title: 'Packet Inspector',
    description: 'Deep packet inspection with protocol decoding and hex dump viewing.',
    path: '/packets',
  },
  {
    title: 'PCAP Analyzer',
    description: 'Upload and analyze PCAP files with detailed statistics and filtering.',
    path: '/pcap',
    badge: 'Beta',
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

export const CATEGORY_LABELS: Record<string, string> = {
  protocol: 'Protocols',
  concept: 'Concepts',
  device: 'Device Types',
  general: 'General',
  navigation: 'Navigation',
  actions: 'Actions',
};
