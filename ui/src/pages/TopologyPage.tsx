import {
  Background,
  BackgroundVariant,
  type Connection,
  Controls,
  type EdgeTypes,
  type Node,
  type NodeTypes,
  Panel,
  ReactFlow,
  type ReactFlowInstance,
  useEdgesState,
  useNodesState,
} from '@xyflow/react';
import { type FC, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import '@xyflow/react/dist/style.css';
import { Network, Radar, RefreshCw } from 'lucide-react';
import { exportTopology, fetchDevices, fetchNeighbors, fetchTopology } from '../api/client';
import type { DeviceSummary } from '../api/types';
import { useAppContext } from '../contexts/AppContext';
import { useApiResource } from '../hooks/useApiResource';
import { useTopologyLayoutPersistence } from '../hooks/useTopologyLayoutPersistence';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { H2, SmallText } from '../ui/Typography';
import {
  ActionsMenu,
  ContextMenu,
  contextMenuItems,
  createEdges,
  DeviceDetailsPanel,
  DeviceNode,
  type DeviceNodeType,
  LAYOUT_MODES,
  type LayoutMode,
  type LinkEdge,
  layoutNodes,
  NeighborsView,
  readSavedLayoutMode,
  TopologyLegend,
  writeSavedLayoutMode,
} from './topology';
import { TrunkEdge } from './topology/TrunkEdge';

/**
 * nodeTypes is the ReactFlow lookup that maps a node.type string onto
 * a React component. We register exactly one — every node is a device.
 * Kept here (not in topology/) because nodeTypes is consumed only by
 * the page-level ReactFlow instance and including it in the barrel
 * would force everyone who imports a single helper to pull DeviceNode.
 */
const nodeTypes: NodeTypes = {
  device: DeviceNode,
};

const edgeTypes: EdgeTypes = {
  trunk: TrunkEdge,
};

// Persistence helpers (positions + layout mode) live in
// ./topology/persistence.ts so this file stays under the 800-line
// file-size red-flag threshold.

/**
 * TopologyPage renders the network graph with the device-node and
 * link-edge presenters defined in ./topology. The page itself owns
 * data fetching (topology + devices + neighbors), state for the
 * legend / minimap / selected device, and the export action.
 */
export const TopologyPage: FC = () => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const navigate = useNavigate();
  const { sessionId } = useAppContext();

  // Fetch topology data from the API with periodic polling
  const {
    data: topology,
    loading: topologyLoading,
    refetch: refetchTopology,
  } = useApiResource(() => fetchTopology(sessionId ?? ''), [sessionId], {
    intervalMs: 15000,
    enabled: sessionId !== null,
  });
  const {
    data: devices,
    loading: devicesLoading,
    refetch: refetchDevices,
  } = useApiResource(() => fetchDevices(sessionId ?? ''), [sessionId], {
    intervalMs: 15000,
    enabled: sessionId !== null,
  });
  const { data: neighbors, refetch: refetchNeighbors } = useApiResource(
    () => fetchNeighbors(sessionId ?? ''),
    [sessionId],
    { intervalMs: 15000, enabled: sessionId !== null },
  );

  const [nodes, setNodes, onNodesChange] = useNodesState<DeviceNodeType>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<LinkEdge>([]);
  const [selectedDevice, setSelectedDevice] = useState<DeviceSummary | null>(null);
  const [showLegend, setShowLegend] = useState(true);
  // Minimap removed in this version — it was always toggle-off by
  // default and operators didn't use it; the ReactFlow Controls
  // (fit-view + zoom buttons) already cover navigation on big graphs.
  // showLabels controls whether TrunkEdge renders its per-side
  // interface labels + the middle VLAN/speed label. On by default
  // for clarity; toggle off via the header to see clean lines only.
  const [showLabels, setShowLabels] = useState(true);
  const [view, setView] = useState<'graph' | 'neighbors'>('graph');
  // Search query in the header — filters which devices show up + their
  // edges. Matched on name (case-insensitive substring).
  const [search, setSearch] = useState('');
  // Type-filter chips — empty Set means "show everything". When the
  // user toggles a type on, only that type renders.
  const [activeTypes, setActiveTypes] = useState<Set<string>>(new Set());
  // Node click selects its neighbourhood: the clicked node + every
  // node connected to it by an edge get full opacity; everything else
  // fades to 15%. Click empty canvas to clear.
  const [focusedNodeId, setFocusedNodeId] = useState<string | null>(null);
  // Tracks which edge the cursor is currently over (ReactFlow's
  // onEdgeMouseEnter / Leave). Propagated down to the TrunkEdge via
  // data.hovered so the custom edge can render the rich tooltip.
  // Lifted up here (rather than useState inside TrunkEdge) so we don't
  // attach mouse handlers to raw SVG paths — that flags
  // lint/a11y/noStaticElementInteractions.
  const [hoveredEdgeId, setHoveredEdgeId] = useState<string | null>(null);
  // Selected layout mode (hierarchical / concentric / grid). Persisted
  // across reloads via localStorage. Hierarchical is the default
  // because rings get hard to read past ~6 devices and the layered
  // dagre layout matches how operators draw networks on a whiteboard.
  const [layoutMode, setLayoutMode] = useState<LayoutMode>(() => readSavedLayoutMode());
  // Node-position persistence (browser-local, debounced writes) — see
  // ../hooks/useTopologyLayoutPersistence.ts. Layout *mode* persistence
  // stays a direct localStorage read/write above; only positions get
  // debounced since they change on every drag.
  const layoutPersistence = useTopologyLayoutPersistence();
  const handleLayoutModeChange = useCallback(
    (mode: LayoutMode) => {
      setLayoutMode(mode);
      writeSavedLayoutMode(mode);
      // A new mode means a fresh position pass — wipe saved drags so
      // the layout algorithm gets a clean canvas. Without this, the
      // user's pre-mode-change positions stick and "Hierarchical" still
      // looks like whatever they had.
      layoutPersistence.resetPositions();
      setNodes((current) => current.map((node) => ({ ...node, position: { x: 0, y: 0 } })));
    },
    [setNodes, layoutPersistence],
  );
  // Session-only "hide this device" set. Cleared by Reset, lost on
  // page reload — we don't persist it because hiding is meant to be a
  // temporary "clean up this view to screenshot" gesture, not a
  // permanent change. Edges referencing a hidden node disappear with
  // it (they get filtered alongside in the build-graph effect).
  const [hiddenDevices, setHiddenDevices] = useState<Set<string>>(new Set());
  // Context-menu state. Discriminated union so render-time exhaustive
  // checks catch missing handlers. Coordinates are in the canvas
  // parent's local space (already adjusted from clientX/Y).
  const [contextMenu, setContextMenu] = useState<
    | null
    | { kind: 'node'; x: number; y: number; nodeId: string }
    | { kind: 'edge'; x: number; y: number; edgeId: string }
    | { kind: 'pane'; x: number; y: number }
  >(null);

  // ReactFlow instance handle — captured on init so we can call
  // fitView() programmatically when the simulation changes
  // underneath us (different config loaded → new device set).
  // Without this, switching templates left the canvas zoomed in
  // on wherever the prior layout had pushed the viewport, and
  // operators interpreted that as a stale render.
  const rfInstance = useRef<ReactFlowInstance<DeviceNodeType, LinkEdge> | null>(null);
  // Bumped by handleResetLayout (declared later) to force the
  // build-graph useEffect to re-run immediately instead of waiting
  // for the next 15-second data poll. Declared up here so the
  // useEffect's dep array can reference it without TDZ trouble.
  const [resetCounter, setResetCounter] = useState(0);
  // Hash of the current device-name set. When this changes we
  // force a fitView so the new map fills the canvas cleanly.
  const lastDeviceSetSig = useRef<string>('');

  // Build graph from API data, merging trunk port links with neighbor-discovered links.
  // resetCounter is intentionally read but not branched on — Reset
  // bumps it to force this effect to re-run with the same upstream
  // data, restoring the fresh-layout positions after Reset clears
  // the cached positions in localStorage.
  useEffect(() => {
    void resetCounter;
    if (!(devices && topology)) {
      return;
    }

    // The daemon returns null arrays when no simulation is loaded; coerce
    // so spread / map / Set initialization don't crash with TypeError.
    const topologyLinks = topology.links ?? [];

    // Start with trunk port links from config
    const allLinks = [...topologyLinks];

    // Merge neighbor-discovered edges that aren't already represented by trunk ports
    if (neighbors && neighbors.length > 0) {
      const existingEdges = new Set(
        topologyLinks.map((l) => [l.source, l.target].sort().join('|')),
      );

      for (const neighbor of neighbors) {
        const key = [neighbor.localDevice, neighbor.remoteDevice].sort().join('|');
        if (!existingEdges.has(key)) {
          existingEdges.add(key);
          allLinks.push({
            source: neighbor.localDevice,
            target: neighbor.remoteDevice,
            label: `${neighbor.protocol} discovery`,
            discovered: true,
          });
        }
      }
    }

    // Build a set of node IDs adjacent to the focused node so we can
    // dim everything else when neighbourhood highlight is active.
    const focused = focusedNodeId;
    const adjacent = new Set<string>();
    if (focused) {
      adjacent.add(focused);
      for (const link of allLinks) {
        if (link.source === focused) adjacent.add(link.target);
        else if (link.target === focused) adjacent.add(link.source);
      }
    }
    const nodeOpacity = (id: string): number => {
      if (!focused) return 1;
      return adjacent.has(id) ? 1 : 0.15;
    };
    const edgeOpacity = (source: string, target: string): number => {
      if (!focused) return 1;
      return adjacent.has(source) && adjacent.has(target) ? 1 : 0.1;
    };

    // Type-filter + search: hide devices that don't match.
    const q = search.trim().toLowerCase();
    const isVisible = (device: DeviceSummary): boolean => {
      if (hiddenDevices.has(device.name)) {
        return false;
      }
      if (activeTypes.size > 0 && !activeTypes.has((device.type || 'unknown').toLowerCase())) {
        return false;
      }
      if (q && !device.name.toLowerCase().includes(q)) {
        return false;
      }
      return true;
    };
    const visibleDevices = devices.filter(isVisible);
    const visibleNames = new Set(visibleDevices.map((d) => d.name));
    const visibleLinks = allLinks.filter(
      (l) => visibleNames.has(l.source) && visibleNames.has(l.target),
    );

    const layoutedNodes = layoutNodes(visibleDevices, visibleLinks, layoutMode);
    const layoutedEdges = createEdges(visibleLinks).map((edge) => ({
      ...edge,
      data: {
        ...edge.data,
        showLabels,
        focusOpacity: edgeOpacity(edge.source, edge.target),
        hovered: hoveredEdgeId === edge.id,
      },
    }));

    // Preserve user-dragged positions across the 15s data poll. For each
    // device that's already on canvas, keep its current position rather
    // than blowing it away with the freshly-computed layout. Brand-new
    // devices that arrived since last render get the fresh layout slot.
    // Also pulls any previously-saved positions out of localStorage so
    // drags survive a page reload.
    setNodes((current) => {
      const positionByName = new Map<string, { x: number; y: number }>();
      for (const node of current) {
        positionByName.set(node.id, node.position);
      }
      const stored = layoutPersistence.loadPositions();
      return layoutedNodes.map((node) => {
        const existing = positionByName.get(node.id);
        const saved = stored[node.id];
        const position = existing ?? saved ?? node.position;
        return {
          ...node,
          position,
          style: { ...node.style, opacity: nodeOpacity(node.id) },
        };
      });
    });
    setEdges(layoutedEdges);
  }, [
    devices,
    topology,
    neighbors,
    showLabels,
    search,
    activeTypes,
    focusedNodeId,
    hoveredEdgeId,
    layoutMode,
    hiddenDevices,
    resetCounter,
    setNodes,
    setEdges,
    layoutPersistence,
  ]);

  // Persist node positions when the user stops dragging so layouts
  // survive page reloads. Keyed by device name; stored (debounced) as
  // a flat {name: {x, y}} map in localStorage — browser-local only.
  const handleNodeDragStop = useCallback(() => {
    const positions: Record<string, { x: number; y: number }> = {};
    for (const node of nodes) {
      positions[node.id] = { x: node.position.x, y: node.position.y };
    }
    layoutPersistence.savePositions(positions);
  }, [nodes, layoutPersistence]);

  // Handle node selection
  const handleNodeClick = useCallback(
    (deviceName: string) => {
      const device = devices?.find((d) => d.name === deviceName);
      setSelectedDevice(device || null);
      // Toggle neighbourhood highlight — clicking the same node twice
      // clears the focus so all nodes/edges return to full opacity.
      setFocusedNodeId((curr) => (curr === deviceName ? null : deviceName));
    },
    [devices],
  );

  // Reset Layout — wipes the saved drag positions + clears focus and
  // search/filter state. The next data-poll effect re-runs layoutNodes
  // from scratch, putting every device on the fresh grid slot.
  // resetCounter declared up with the other refs/state. handleResetLayout
  // bumps it to force the build-graph useEffect to re-run immediately
  // instead of waiting for the next 15-second data poll. The previous
  // Reset implementation set positions to {x:0, y:0} hoping the effect
  // would notice — but (0,0) is a valid coordinate, so the effect's
  // "existing ?? saved ?? fresh" fallback kept the (0,0) and visibly
  // did nothing.
  const handleResetLayout = useCallback(() => {
    layoutPersistence.resetPositions();
    setFocusedNodeId(null);
    setSearch('');
    setActiveTypes(new Set());
    setHiddenDevices(new Set());
    setNodes([]);
    setEdges([]);
    setResetCounter((n) => n + 1);
    // Re-frame the canvas after the fresh layout settles.
    window.setTimeout(() => {
      rfInstance.current?.fitView({ padding: 0.2, duration: 400 });
    }, 150);
  }, [setNodes, setEdges, layoutPersistence]);

  // The set of unique device types currently in the graph — drives the
  // filter chip row. Sorted for stable button order.
  const availableTypes = useMemo(() => {
    if (!devices) return [];
    const set = new Set<string>();
    for (const d of devices) set.add((d.type || 'unknown').toLowerCase());
    return [...set].sort();
  }, [devices]);

  const toggleType = useCallback((type: string) => {
    setActiveTypes((curr) => {
      const next = new Set(curr);
      if (next.has(type)) next.delete(type);
      else next.add(type);
      return next;
    });
  }, []);

  // Hide a device by name. Session-only — Reset re-shows everything.
  const hideDevice = useCallback((name: string) => {
    setHiddenDevices((curr) => {
      const next = new Set(curr);
      next.add(name);
      return next;
    });
  }, []);

  // Copy helper — silently no-ops on clipboard-API errors (the most
  // common failure is a non-secure context, which we can't fix from
  // here). The action menu items intentionally don't surface a toast
  // because right-click → copy is a fast, low-stakes gesture.
  const copyToClipboard = useCallback((text: string) => {
    if (typeof navigator !== 'undefined' && navigator.clipboard) {
      void navigator.clipboard.writeText(text).catch(() => {
        // Ignore — secure-context or permissions issue. The user can
        // fall back to manual select.
      });
    }
  }, []);

  // Context-menu open helpers. ReactFlow calls these with the native
  // mouse event + the node/edge object; we adjust coordinates into
  // the canvas-parent's local space so the menu pops where the
  // pointer is regardless of page scroll position.
  // Context-menu handlers pass clientX/Y straight through to a
  // position:fixed menu. Earlier attempts offset by the event's
  // currentTarget rect — but ReactFlow's callbacks set currentTarget
  // to the node/edge/pane element itself (not the canvas container),
  // so the subtraction produced node-relative coords and the menu
  // popped off-screen. Viewport-fixed is simpler and matches how
  // every OS-native context menu behaves.
  const handleNodeContextMenu = useCallback((event: React.MouseEvent, node: Node) => {
    event.preventDefault();
    setContextMenu({ kind: 'node', x: event.clientX, y: event.clientY, nodeId: node.id });
  }, []);

  const handleEdgeContextMenu = useCallback((event: React.MouseEvent, edge: { id: string }) => {
    event.preventDefault();
    setContextMenu({ kind: 'edge', x: event.clientX, y: event.clientY, edgeId: edge.id });
  }, []);

  const handlePaneContextMenu = useCallback((event: React.MouseEvent | MouseEvent) => {
    event.preventDefault();
    const me = event as MouseEvent;
    setContextMenu({ kind: 'pane', x: me.clientX, y: me.clientY });
  }, []);

  const closeContextMenu = useCallback(() => setContextMenu(null), []);

  // Update node data with click handler
  useEffect(() => {
    setNodes((nds) =>
      nds.map((node) => ({
        ...node,
        data: {
          ...node.data,
          onClick: handleNodeClick,
        },
      })),
    );
  }, [handleNodeClick, setNodes]);

  // When the underlying simulation changes (different config →
  // different device set), force a fitView so the canvas re-frames
  // around the new graph instead of staying zoomed in on whatever
  // the previous layout had pushed the viewport to. Detected via a
  // sorted-name signature hash; same names = same sim = no re-fit.
  useEffect(() => {
    if (!devices || devices.length === 0) return;
    const sig = devices
      .map((d) => d.name)
      .sort()
      .join('|');
    if (sig === lastDeviceSetSig.current) return;
    lastDeviceSetSig.current = sig;
    // Defer one tick so layoutNodes has populated the new positions
    // before fitView measures them.
    const t = window.setTimeout(() => {
      rfInstance.current?.fitView({ padding: 0.2, duration: 400 });
    }, 50);
    return () => window.clearTimeout(t);
  }, [devices]);

  // onConnect is a no-op: edges come from the backend and are not user-editable
  const onConnect = useCallback((_params: Connection) => {
    // Intentionally empty - topology edges are derived from device configs
    // and neighbor discovery; user-drawn edges would not persist.
  }, []);

  // Export topology as JSON
  const handleExport = useCallback(() => {
    const exportData = {
      nodes: nodes.map((n) => ({
        name: n.id,
        type: n.data.type,
        ips: n.data.ips,
        protocols: n.data.protocols,
        position: n.position,
      })),
      edges: edges.map((e) => ({
        source: e.source,
        target: e.target,
        label: e.label,
        data: e.data,
      })),
    };
    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `niac-topology-${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [nodes, edges]);

  const [exportError, setExportError] = useState<string | null>(null);

  // Server-side topology export (DOT / GraphML). Unlike the client-side JSON
  // snapshot above, this fetches the daemon's rendered topology, so the output
  // reflects what the running simulation actually sees — suitable for feeding
  // into Graphviz / yEd / gephi.
  const exportServerFormat = useCallback(
    async (format: 'dot' | 'graphml', extension: string, mimeType: string) => {
      setExportError(null);
      try {
        const content = await exportTopology(format);
        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = `niac-topology-${new Date().toISOString().slice(0, 10)}.${extension}`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);
      } catch (err) {
        setExportError((err as Error).message);
      }
    },
    [],
  );

  const handleExportDOT = useCallback(
    () => void exportServerFormat('dot', 'dot', 'text/vnd.graphviz'),
    [exportServerFormat],
  );
  const handleExportGraphML = useCallback(
    () => void exportServerFormat('graphml', 'graphml', 'application/xml'),
    [exportServerFormat],
  );

  // Refresh data by refetching from the API. We bump the reset
  // counter so the build-graph effect re-runs even when the data
  // payload is identical — otherwise repeat clicks looked like
  // no-ops because devices/topology references stayed the same.
  // Critical: await ALL refetches before scheduling fitView. Earlier
  // versions used a fixed 150ms setTimeout and on slow networks
  // the fitView fired against stale node positions, then the new
  // data landed and the canvas snapped to a different frame —
  // looked broken. requestAnimationFrame gives React one paint to
  // commit the new layout before we re-frame.
  const handleRefresh = useCallback(() => {
    setResetCounter((n) => n + 1);
    Promise.all([refetchTopology(), refetchDevices(), refetchNeighbors()])
      .catch(() => {
        // Refetch errors surface through the per-hook error states;
        // nothing to do here besides making sure we still try the
        // refit (better than freezing on a transient network blip).
      })
      .finally(() => {
        window.requestAnimationFrame(() => {
          rfInstance.current?.fitView({ padding: 0.2, duration: 300 });
        });
      });
  }, [refetchTopology, refetchDevices, refetchNeighbors]);

  const loading = topologyLoading || devicesLoading;

  // PNG export. Captures the ReactFlow viewport via the .react-flow
  // root element; html-to-image walks the DOM, inlines computed
  // styles, and rasterises to PNG. We snapshot at 2× pixel ratio so
  // the image scales cleanly on Retina/4K screens.
  const handleExportPNG = useCallback(async () => {
    setExportError(null);
    const root = document.querySelector<HTMLElement>('.react-flow');
    if (!root) {
      setExportError('Could not find the topology canvas to export.');
      return;
    }
    try {
      const { toPng } = await import('html-to-image');
      const dataUrl = await toPng(root, {
        pixelRatio: 2,
        backgroundColor: '#0a0a0a',
        cacheBust: true,
        // Skip the ReactFlow controls + minimap chrome — viewers want
        // the diagram, not the UI scaffolding.
        filter: (node) =>
          !(node instanceof HTMLElement) || !node.classList?.contains?.('react-flow__panel'),
      });
      const link = document.createElement('a');
      link.href = dataUrl;
      link.download = `niac-topology-${new Date().toISOString().slice(0, 10)}.png`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    } catch (err) {
      setExportError((err as Error).message || 'Failed to export topology as PNG.');
    }
  }, []);

  return (
    <div className="stack-xl">
      {/* Header Card */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent>
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-comfortable">
            <div className="flex items-center gap-default">
              <div className="pad-xs rounded-lg bg-status-info/20">
                <Network className="w-6 h-6 text-status-info" />
              </div>
              <div>
                <H2>{t('topology.page.networkTopologyTitle')}</H2>
                <SmallText className="text-text-muted">
                  {tCommon('plurals.deviceCount', { count: devices?.length || 0 })} |{' '}
                  {t('topology.page.connectionCount', { count: topology?.links?.length || 0 })}
                </SmallText>
              </div>
            </div>

            <div className="flex flex-wrap items-center gap-compact">
              <div
                className="inline-flex rounded-lg border border-surface-border bg-bg-base/40 p-0.5"
                role="tablist"
                aria-label={t('topology.header.viewTabsAriaLabel')}
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={view === 'graph'}
                  onClick={() => setView('graph')}
                  title={t('topology.header.graphTabTitle')}
                  className={`flex items-center gap-1.5 rounded px-3 py-compact text-xs font-medium transition-colors ${
                    view === 'graph'
                      ? 'bg-status-info/20 text-status-info'
                      : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
                  }`}
                >
                  <Network className="w-3.5 h-3.5" />
                  {t('topology.header.graphTabLabel')}
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={view === 'neighbors'}
                  onClick={() => setView('neighbors')}
                  title={t('topology.header.neighborsTabTitle')}
                  className={`flex items-center gap-1.5 rounded px-3 py-compact text-xs font-medium transition-colors ${
                    view === 'neighbors'
                      ? 'bg-status-info/20 text-status-info'
                      : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
                  }`}
                >
                  <Radar className="w-3.5 h-3.5" />
                  {t('topology.header.neighborsTabLabel')}
                </button>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleRefresh}
                leftIcon={<RefreshCw className="w-4 h-4" />}
                disabled={loading}
              >
                {tCommon('buttons.refresh')}
              </Button>
              {view === 'graph' && (
                <>
                  {/* Reset is always-visible — it's a recovery action
                      (snap drags / filters back to defaults) and
                      operators reach for it often enough that hiding
                      it inside a menu got friction reports. */}
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleResetLayout}
                    disabled={nodes.length === 0}
                    title={t('topology.header.resetButtonTitle')}
                    data-testid="topology-reset-layout-button"
                  >
                    {t('topology.header.resetButtonLabel')}
                  </Button>
                  <Button
                    variant={showLabels ? 'outline' : 'ghost'}
                    size="sm"
                    onClick={() => setShowLabels(!showLabels)}
                    title={t('topology.header.labelsButtonTitle')}
                  >
                    {t('topology.header.labelsButtonLabel')}
                  </Button>
                  {/* Export is the only menu — PNG vs JSON is a real
                      two-option choice, but Reset + Labels + Layout
                      are direct buttons / pills for fast iteration. */}
                  <ActionsMenu
                    disabled={nodes.length === 0}
                    onExportPNG={handleExportPNG}
                    onExportJSON={handleExport}
                    onExportDOT={handleExportDOT}
                    onExportGraphML={handleExportGraphML}
                  />
                  {exportError && (
                    <SmallText role="alert" className="text-status-error">
                      {exportError}
                    </SmallText>
                  )}
                  {/* Layout-mode picker — one pill per mode. The active
                      pill is highlighted; clicking another re-runs the
                      layout and persists the choice. Hierarchical is the
                      default (rings get hard to read past ~6 devices). */}
                  {/* role/aria-pressed pattern (not radiogroup) — biome's
                      a11y rule says role="radio" wants a real <input
                      type="radio">. The interaction here is the same as
                      the existing Graph/Neighbors view picker above,
                      which uses tablist; we use aria-pressed instead so
                      either pattern (radio or toggle) reads cleanly. */}
                  <div className="inline-flex rounded-lg border border-surface-border bg-bg-base/40 p-0.5">
                    {LAYOUT_MODES.map((entry) => {
                      const active = layoutMode === entry.mode;
                      return (
                        <button
                          key={entry.mode}
                          type="button"
                          aria-pressed={active}
                          aria-label={t('topology.header.layoutModeAriaLabel', {
                            label: entry.label,
                          })}
                          title={entry.description}
                          onClick={() => handleLayoutModeChange(entry.mode)}
                          className={`rounded px-2.5 py-compact text-xs font-medium transition-colors ${
                            active
                              ? 'bg-status-info/20 text-status-info'
                              : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
                          }`}
                        >
                          {entry.label}
                        </button>
                      );
                    })}
                  </div>
                  {/* Unobtrusive reminder that dragged positions live in
                      this browser's localStorage only — no backend
                      route, so a different browser/profile starts from
                      the default layout. Hidden below `lg` to keep the
                      toolbar from wrapping on narrow viewports. */}
                  <SmallText
                    className="hidden lg:inline text-text-muted"
                    title={t('topology.header.layoutPersistenceHint')}
                    data-testid="topology-layout-persistence-note"
                  >
                    {t('topology.header.layoutPersistenceNote')}
                  </SmallText>
                </>
              )}
            </div>
          </div>

          {/* Second header row: search + type-filter chips. Only shown
              for the graph view — the Neighbors table has its own
              filtering UI. Stays inside the header card so the toolbar
              groups visually with the title/buttons row above. */}
          {view === 'graph' && (
            <div className="mt-heading flex flex-col gap-compact sm:flex-row sm:items-center sm:gap-comfortable">
              <input
                type="search"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('topology.header.searchPlaceholder')}
                aria-label={t('topology.header.searchAriaLabel')}
                className="w-full sm:w-64 rounded-md border border-surface-border bg-bg-base/40 px-3 py-compact-md text-xs text-text-primary placeholder:text-text-muted focus:border-status-info/40 focus:outline-none focus:ring-1 focus:ring-status-info/30"
              />
              {availableTypes.length > 0 && (
                <div className="flex flex-wrap items-center gap-1.5">
                  <SmallText className="text-text-muted">
                    {t('topology.header.typesLabel')}
                  </SmallText>
                  {availableTypes.map((type) => {
                    const active = activeTypes.has(type);
                    return (
                      <button
                        key={type}
                        type="button"
                        onClick={() => toggleType(type)}
                        aria-pressed={active}
                        title={
                          active
                            ? t('topology.header.typeFilterHideTitle', { type })
                            : t('topology.header.typeFilterShowTitle', { type })
                        }
                        className={`rounded-full border px-2.5 py-0.5 text-[11px] font-medium capitalize transition-colors ${
                          active
                            ? 'border-status-info/40 bg-status-info/20 text-status-info'
                            : 'border-surface-border bg-bg-base/40 text-text-muted hover:bg-surface-hover hover:text-text-primary'
                        }`}
                      >
                        {type}
                      </button>
                    );
                  })}
                  {activeTypes.size > 0 && (
                    <button
                      type="button"
                      onClick={() => setActiveTypes(new Set())}
                      title={t('topology.header.clearTypeFiltersTitle')}
                      className="rounded-full px-cell py-0.5 text-[11px] font-medium text-text-muted hover:text-text-primary underline-offset-2 hover:underline"
                    >
                      {tCommon('buttons.clear')}
                    </button>
                  )}
                </div>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Topology Visualization — sized to fill the available viewport
          minus the page chrome (header card + breadcrumbs + page header
          ≈ 240 px on this layout). Gives much more room for the graph
          on standard 1080p+ displays. */}
      {view === 'graph' && (
        <Card className="border-surface-border bg-bg-surface/70 overflow-hidden">
          <div className="h-[calc(100vh-260px)] min-h-[520px] relative">
            {loading ? (
              <div className="absolute inset-0 flex-center">
                <div className="flex flex-col items-center gap-default">
                  <RefreshCw className="w-8 h-8 text-brand-accent animate-spin" />
                  <SmallText className="text-text-muted">
                    {t('topology.page.loadingTopology')}
                  </SmallText>
                </div>
              </div>
            ) : nodes.length === 0 ? (
              <div className="absolute inset-0 flex-center">
                <div className="text-center">
                  <Network className="w-16 h-16 text-text-disabled mx-auto mb-content" />
                  <p className="text-text-muted mb-2">{t('topology.page.noTopologyData')}</p>
                  <SmallText className="text-text-muted">
                    {t('topology.page.noConnectionsHint')}
                  </SmallText>
                </div>
              </div>
            ) : (
              <>
                {edges.length === 0 && (
                  // z-50 wins over ReactFlow's internal Panel chrome and
                  // pointer-events-none lets pan/zoom drag through the
                  // banner so it doesn't trap clicks on the canvas below.
                  <div className="pointer-events-none absolute top-0 left-0 right-0 z-50 bg-status-warning/60 border-b border-status-warning/40 px-4 py-row text-center backdrop-blur-sm">
                    <SmallText className="text-status-warning">
                      {t('topology.page.noLinksBannerAddPrefix')}{' '}
                      <code className="text-status-warning">trunk_ports:</code>{' '}
                      {t('topology.page.noLinksBannerOr')}{' '}
                      <code className="text-status-warning">port_channels:</code>{' '}
                      {t('topology.page.noLinksBannerSuffix')}{' '}
                      <strong>{t('topology.header.neighborsTabLabel')}</strong>{' '}
                      {t('topology.page.noLinksBannerPageSuffix')}
                    </SmallText>
                  </div>
                )}
                <ReactFlow
                  nodes={nodes}
                  edges={edges}
                  onNodesChange={onNodesChange}
                  onEdgesChange={onEdgesChange}
                  onConnect={onConnect}
                  onNodeDragStop={handleNodeDragStop}
                  onEdgeMouseEnter={(_, edge) => setHoveredEdgeId(edge.id)}
                  onEdgeMouseLeave={() => setHoveredEdgeId(null)}
                  onNodeContextMenu={handleNodeContextMenu}
                  onEdgeContextMenu={handleEdgeContextMenu}
                  onPaneContextMenu={handlePaneContextMenu}
                  // Left-click anywhere closes any open context menu;
                  // right-clicking again on a different target opens
                  // its menu in place.
                  onPaneClick={closeContextMenu}
                  onNodeClick={(_, node) => {
                    closeContextMenu();
                    handleNodeClick(node.id);
                  }}
                  onInit={(instance) => {
                    rfInstance.current = instance;
                  }}
                  nodeTypes={nodeTypes}
                  edgeTypes={edgeTypes}
                  fitView={true}
                  fitViewOptions={{ padding: 0.2 }}
                  minZoom={0.2}
                  maxZoom={2}
                  defaultEdgeOptions={{
                    type: 'smoothstep',
                  }}
                  proOptions={{ hideAttribution: true }}
                >
                  <Background
                    variant={BackgroundVariant.Dots}
                    gap={20}
                    size={1}
                    color="#ffffff0d"
                  />
                  <Controls showZoom={true} showFitView={true} showInteractive={false} />

                  {/* Legend Panel — only meaningful when there are edges
                      to colour. With zero edges the link-speed key is
                      just noise. */}
                  {edges.length > 0 && (
                    <Panel position="top-left">
                      <TopologyLegend
                        show={showLegend}
                        onToggle={() => setShowLegend(!showLegend)}
                      />
                    </Panel>
                  )}

                  {/* Selected Device Panel */}
                  {selectedDevice && (
                    <DeviceDetailsPanel
                      device={selectedDevice}
                      onClose={() => setSelectedDevice(null)}
                      onEdit={(device) => {
                        navigate(`/device-config/${device.name}`);
                      }}
                    />
                  )}
                </ReactFlow>
                {/* Right-click context menu. Coordinates are in the
                    canvas-parent's local space (already offset from
                    the bounding rect in the open handlers). */}
                {contextMenu && (
                  <ContextMenu
                    x={contextMenu.x}
                    y={contextMenu.y}
                    onClose={closeContextMenu}
                    items={contextMenuItems(contextMenu, {
                      t,
                      edges,
                      hideDevice,
                      navigate,
                      setFocusedNodeId,
                      setSelectedDevice,
                      devices,
                      handleResetLayout,
                      handleExport,
                      copyToClipboard,
                    })}
                  />
                )}
              </>
            )}
          </div>
        </Card>
      )}

      {/* Neighbors view — replaces the standalone /neighbors page */}
      {view === 'neighbors' && <NeighborsView />}
    </div>
  );
};

export default TopologyPage;
