// Copyright (c) 2025 Mustard Seed Networks. All rights reserved.

/**
 * =============================================================================
 * DEVICE COLORS - NIAC specific
 * =============================================================================
 *
 * Device type colors - accent colors for network device visualization.
 * Protocol colors - for packet/protocol identification.
 * Link speed colors - for topology visualization.
 */

/**
 * Device type colors - accent colors for network device visualization
 */
export const deviceColor = {
  router: {
    icon: 'text-blue-500',
    badge: 'bg-blue-500/20 text-blue-400 border border-blue-500/30',
    border: 'border-blue-500/30',
    bg: 'bg-blue-500',
  },
  switch: {
    icon: 'text-emerald-500',
    badge: 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30',
    border: 'border-emerald-500/30',
    bg: 'bg-emerald-500',
  },
  firewall: {
    icon: 'text-red-500',
    badge: 'bg-red-500/20 text-red-400 border border-red-500/30',
    border: 'border-red-500/30',
    bg: 'bg-red-500',
  },
  server: {
    icon: 'text-orange-500',
    badge: 'bg-orange-500/20 text-orange-400 border border-orange-500/30',
    border: 'border-orange-500/30',
    bg: 'bg-orange-500',
  },
  workstation: {
    icon: 'text-gray-400',
    badge: 'bg-gray-500/20 text-gray-400 border border-gray-500/30',
    border: 'border-gray-500/30',
    bg: 'bg-gray-500',
  },
  ap: {
    icon: 'text-purple-500',
    badge: 'bg-purple-500/20 text-purple-400 border border-purple-500/30',
    border: 'border-purple-500/30',
    bg: 'bg-purple-500',
  },
  iot: {
    icon: 'text-teal-500',
    badge: 'bg-teal-500/20 text-teal-400 border border-teal-500/30',
    border: 'border-teal-500/30',
    bg: 'bg-teal-500',
  },
  unknown: {
    icon: 'text-gray-500',
    badge: 'bg-gray-500/20 text-gray-400 border border-gray-500/30',
    border: 'border-gray-500/30',
    bg: 'bg-gray-500',
  },
} as const;

/**
 * Protocol colors - for packet/protocol identification
 */
export const protocolColor = {
  arp: { icon: 'text-emerald-500', badge: 'bg-emerald-500/20 text-emerald-400' },
  icmp: { icon: 'text-blue-500', badge: 'bg-blue-500/20 text-blue-400' },
  dns: { icon: 'text-purple-500', badge: 'bg-purple-500/20 text-purple-400' },
  dhcp: { icon: 'text-orange-500', badge: 'bg-orange-500/20 text-orange-400' },
  snmp: { icon: 'text-teal-500', badge: 'bg-teal-500/20 text-teal-400' },
  lldp: { icon: 'text-pink-500', badge: 'bg-pink-500/20 text-pink-400' },
  cdp: { icon: 'text-yellow-500', badge: 'bg-yellow-500/20 text-yellow-400' },
  http: { icon: 'text-indigo-500', badge: 'bg-indigo-500/20 text-indigo-400' },
  tcp: { icon: 'text-cyan-500', badge: 'bg-cyan-500/20 text-cyan-400' },
  udp: { icon: 'text-violet-500', badge: 'bg-violet-500/20 text-violet-400' },
} as const;

/**
 * Link speed colors - for topology visualization
 */
export const linkSpeedColor = {
  '10m': 'text-gray-400',
  '100m': 'text-emerald-500',
  '1g': 'text-blue-500',
  '10g': 'text-purple-500',
  '25g': 'text-orange-500',
  '40g': 'text-pink-500',
  '100g': 'text-yellow-500',
  trunk: 'text-cyan-500',
} as const;
