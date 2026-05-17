import type { useNavigate } from 'react-router-dom';
import type { DeviceSummary } from '../../api/types';
import type { ContextMenuItem } from './ContextMenu';
import type { LinkEdge } from './types';

/**
 * Discriminated union of context-menu targets. Render-time exhaustive
 * checks catch missing handlers in contextMenuItems() below.
 */
export type ContextMenuTarget =
  | { kind: 'node'; nodeId: string }
  | { kind: 'edge'; edgeId: string }
  | { kind: 'pane' };

/**
 * ContextMenuCtx is the bag of callbacks + state a menu item handler
 * needs to actually do its work — held by TopologyPage and passed in
 * each time the menu opens so we don't have to plumb ten props
 * through ContextMenu itself.
 */
export interface ContextMenuCtx {
  edges: LinkEdge[];
  hideDevice: (name: string) => void;
  navigate: ReturnType<typeof useNavigate>;
  setFocusedNodeId: (next: ((curr: string | null) => string | null) | string | null) => void;
  setSelectedDevice: (device: DeviceSummary | null) => void;
  devices: DeviceSummary[] | null;
  handleResetLayout: () => void;
  handleExport: () => void;
  copyToClipboard: (text: string) => void;
}

/**
 * contextMenuItems builds the action list for a given menu target.
 * Factored out of the render path so the JSX stays tight and so each
 * menu's items are produced in one place rather than scattered inline.
 *
 * Item presence is data-driven — e.g. "Edit YAML" only appears when
 * the daemon knows the device (the menu was opened on a node whose
 * name is in the devices list), and the VLAN action only appears
 * when the edge carries vlan data.
 *
 * Lives in its own file so TopologyPage.tsx stays under the 800-line
 * file-size red-flag threshold.
 */
export function contextMenuItems(menu: ContextMenuTarget, ctx: ContextMenuCtx): ContextMenuItem[] {
  switch (menu.kind) {
    case 'node':
      return nodeMenuItems(menu.nodeId, ctx);
    case 'edge':
      return edgeMenuItems(menu.edgeId, ctx);
    default:
      return paneMenuItems(ctx);
  }
}

function nodeMenuItems(nodeId: string, ctx: ContextMenuCtx): ContextMenuItem[] {
  const device = ctx.devices?.find((d) => d.name === nodeId);
  return [
    {
      key: 'view',
      label: 'View details',
      onSelect: () => {
        if (device) ctx.setSelectedDevice(device);
      },
      disabled: !device,
    },
    {
      key: 'edit',
      label: 'Edit YAML',
      hint: 'devices →',
      onSelect: () => ctx.navigate(`/device-config/${nodeId}`),
    },
    {
      key: 'focus',
      label: 'Focus neighbourhood',
      onSelect: () =>
        ctx.setFocusedNodeId((curr: string | null) => (curr === nodeId ? null : nodeId)),
    },
    {
      key: 'copy-name',
      label: 'Copy name',
      onSelect: () => ctx.copyToClipboard(nodeId),
      separatorBefore: true,
    },
    {
      key: 'hide',
      label: 'Hide from view',
      hint: 'until Reset',
      destructive: true,
      onSelect: () => ctx.hideDevice(nodeId),
      separatorBefore: true,
    },
  ];
}

function edgeMenuItems(edgeId: string, ctx: ContextMenuCtx): ContextMenuItem[] {
  const edge = ctx.edges.find((e) => e.id === edgeId);
  const ifacePair =
    edge?.data?.sourceInterface || edge?.data?.targetInterface
      ? `${edge?.source}:${edge?.data?.sourceInterface ?? '?'} ↔ ${edge?.target}:${edge?.data?.targetInterface ?? '?'}`
      : `${edge?.source} ↔ ${edge?.target}`;
  const items: ContextMenuItem[] = [
    {
      key: 'copy-pair',
      label: 'Copy device/interface pair',
      onSelect: () => ctx.copyToClipboard(ifacePair),
    },
  ];
  // Only show VLAN actions when the edge actually carries VLAN data —
  // keeps the menu short for access links / discovered edges that
  // have nothing VLAN-shaped to filter on.
  if (edge?.data?.vlans && edge.data.vlans.length > 0) {
    const vlanList = edge.data.vlans.join(',');
    items.push({
      key: 'copy-vlans',
      label: `Copy VLAN list (${edge.data.vlans.length})`,
      onSelect: () => ctx.copyToClipboard(vlanList),
    });
  }
  return items;
}

function paneMenuItems(ctx: ContextMenuCtx): ContextMenuItem[] {
  return [
    {
      key: 'export',
      label: 'Export topology JSON',
      onSelect: ctx.handleExport,
    },
    {
      key: 'reset',
      label: 'Reset layout',
      hint: 'clears drags / filters',
      destructive: true,
      onSelect: ctx.handleResetLayout,
      separatorBefore: true,
    },
  ];
}
