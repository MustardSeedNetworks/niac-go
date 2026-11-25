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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { normalizeDeviceType } from '../constants/device-types';
import { useDeviceTypeLabels } from '../hooks/useDeviceTypeLabels';
import '@xyflow/react/dist/style.css';
import { Network, Radar, RefreshCw } from 'lucide-react';
import type { DeviceSummary } from '../api/types';
import { useAppContext, useAppState } from '../contexts/AppContext';
import { useTopologyResource } from '../hooks/usePageResources';
import { useResourceError } from '../hooks/useResourceError';
import { useTopologyLayoutPersistence } from '../hooks/useTopologyLayoutPersistence';
import { Button } from '../shared/Button';
import { Tooltip } from '../shared/Tooltip';
import { ActionButton } from '../ui/ActionButton';
import { Card, CardContent } from '../ui/Card';
import { H2, SmallText } from '../ui/Typography';
import {
  ActionsMenu,
  ContextMenu,
  contextMenuItems,
  DeviceDetailsPanel,
  DeviceNode,
  type DeviceNodeType,
  LAYOUT_MODES,
  type LayoutMode,
  type LinkEdge,
  NeighborsView,
  readSavedLayoutMode,
  TierBands,
  TopologyLegend,
  useTierBands,
  useTopologyExport,
  writeSavedLayoutMode,
} from './topology';
import { CanvasState } from './topology/CanvasState';
import { neighbourhood, sitesOf, type TopologyFilter } from './topology/graph';
import { decorateEdges } from './topology/labelPlacement';
import { reframeAfterPaint } from './topology/reframe';
import { TrunkEdge } from './topology/TrunkEdge';
import { useTopologyLayout } from './topology/useTopologyLayout';

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

/**
 * TopologyPage renders the network graph with the device-node and
 * link-edge presenters defined in ./topology. The page itself owns
 * data fetching (topology + devices + neighbors), state for the
 * legend / minimap / selected device, and the export action.
 */
export function TopologyPage() {
  const { t } = useTranslation('pages');
  const deviceTypeLabels = useDeviceTypeLabels();
  const { t: tCommon } = useTranslation('common');
  const navigate = useNavigate();
  const { sessionId } = useAppContext();

  // Runtime reads name their session; nothing to read before one runs.
  const {
    data: topology,
    loading: topologyLoading,
    error: topologyError,
    refetch: refetchTopology,
  } = useTopologyResource(sessionId);
  const {
    data: devices,
    loading: devicesLoading,
    error: devicesError,
    refetch: refetchDevices,
  } = useAppState('devices');
  // Neighbours enrich the graph rather than draw it, so a failure there is a
  // toast, not a blocked canvas.
  const {
    data: neighbors,
    error: neighborsError,
    refetch: refetchNeighbors,
  } = useAppState('neighbors');
  useResourceError(neighborsError, t('topology.page.neighborsFailed'));

  const [nodes, setNodes, onNodesChange] = useNodesState<DeviceNodeType>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<LinkEdge>([]);
  const [selectedDevice, setSelectedDevice] = useState<DeviceSummary | null>(null);
  const [showLegend, setShowLegend] = useState(true);
  // showLabels controls whether TrunkEdge renders its per-side interface
  // labels + the middle VLAN/speed label. On by default for clarity.
  const [showLabels, setShowLabels] = useState(true);
  const [view, setView] = useState<'graph' | 'neighbors'>('graph');
  // Filters devices + their edges by name, case-insensitive substring.
  const [search, setSearch] = useState('');
  // Empty Set means "show everything"; toggling a type renders only it.
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

  const { tiers, extent: tierExtent } = useTierBands(nodes, layoutMode);
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
  // it (filterGraph drops them with the device).
  const [hiddenDevices, setHiddenDevices] = useState<Set<string>>(new Set());
  // A pack stamps each device with its site. One layout of every site at once
  // is unreadably wide at enterprise scale, so the canvas can show one site.
  const [activeSite, setActiveSite] = useState<string | null>(null);
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
  // Without this, switching scenarios left the canvas zoomed in
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

  // The filter is one memoised object so its identity changes only when a
  // filter does, which is what lets useTopologyLayout skip dagre otherwise.
  const filter = useMemo<TopologyFilter>(
    () => ({ hidden: hiddenDevices, types: activeTypes, search, site: activeSite }),
    [hiddenDevices, activeTypes, search, activeSite],
  );
  // resetCounter is the layout revision: Reset and Refresh bump it to force a
  // fresh layout over data that has not changed.
  const layout = useTopologyLayout({
    devices,
    topology,
    neighbors,
    filter,
    layoutMode,
    revision: resetCounter,
  });

  // Carry position, selection and measured size across the 15s data poll:
  // keyboard moves act on the selection, and React Flow hides a node with no
  // size, blanking the canvas and dropping focus. New devices get the fresh
  // layout slot; saved positions come from localStorage so drags survive a
  // page reload.
  useEffect(() => {
    if (!layout) {
      return;
    }
    // Everything outside the focused node's neighbourhood fades, nodes and
    // edges alike.
    const adjacent = focusedNodeId ? neighbourhood(layout.links, focusedNodeId) : null;
    setNodes((current) => {
      const currentByName = new Map(current.map((node) => [node.id, node]));
      const stored = layoutPersistence.loadPositions();
      return layout.nodes.map((node) => {
        const existing = currentByName.get(node.id);
        const saved = stored[node.id];
        const position = existing?.position ?? saved ?? node.position;
        return {
          ...node,
          position,
          selected: existing?.selected,
          measured: existing?.measured,
          style: { ...node.style, opacity: !adjacent || adjacent.has(node.id) ? 1 : 0.15 },
          // DeviceNode's button is the named tab stop. The wrapper leaves the
          // tab order but stays focusable, so its select-and-move key handler
          // still hears keys bubbling up from the button.
          domAttributes: { tabIndex: -1 },
        };
      });
    });
  }, [layout, focusedNodeId, setNodes, layoutPersistence]);

  // Edges re-decorate on hover and focus without re-placing a label.
  useEffect(() => {
    if (!layout) {
      return;
    }
    const adjacent = focusedNodeId ? neighbourhood(layout.links, focusedNodeId) : null;
    setEdges(
      decorateEdges(layout.edges, layout.placements, {
        showLabels,
        focusOpacity: (source, target) =>
          !adjacent || (adjacent.has(source) && adjacent.has(target)) ? 1 : 0.1,
        hoveredEdgeId,
      }),
    );
  }, [layout, focusedNodeId, showLabels, hoveredEdgeId, setEdges]);

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
    setActiveSite(null);
    setNodes([]);
    setEdges([]);
    setResetCounter((n) => n + 1);
    reframeAfterPaint(() => rfInstance.current?.fitView({ padding: 0.2, duration: 400 }));
  }, [setNodes, setEdges, layoutPersistence]);

  // The set of unique device types currently in the graph — drives the
  // filter chip row. Sorted for stable button order.
  const availableTypes = useMemo(() => {
    if (!devices) return [];
    const set = new Set<string>();
    // Normalised, so `ap` and `access-point` — the same device, two spellings
    // the schema both accepts — produce one chip rather than two identical ones.
    for (const d of devices) set.add(normalizeDeviceType(d.type));
    return [...set].sort((a, b) => a.localeCompare(b));
  }, [devices]);

  const availableSites = useMemo(() => sitesOf(devices ?? []), [devices]);

  // A different site is a different graph, so the nodes already on canvas must
  // not carry their old positions into it.
  const handleSiteChange = useCallback(
    (site: string | null) => {
      setActiveSite(site);
      setNodes([]);
      reframeAfterPaint(() => rfInstance.current?.fitView({ padding: 0.2, duration: 300 }));
    },
    [setNodes],
  );

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
      .sort((a, b) => a.localeCompare(b))
      .join('|');
    if (sig === lastDeviceSetSig.current) return;
    lastDeviceSetSig.current = sig;
    // Defer one tick so layoutNodes has populated the new positions
    // before fitView measures them.
    const cancelReframe = reframeAfterPaint(() =>
      rfInstance.current?.fitView({ padding: 0.2, duration: 400 }),
    );
    return cancelReframe;
  }, [devices]);

  const {
    error: exportError,
    exportJSON: handleExport,
    exportDOT: handleExportDOT,
    exportGraphML: handleExportGraphML,
    exportPNG: handleExportPNG,
  } = useTopologyExport(nodes, edges);

  // onConnect is a no-op: edges come from the backend and are not user-editable
  const onConnect = useCallback((_params: Connection) => {
    // Intentionally empty - topology edges are derived from device configs
    // and neighbor discovery; user-drawn edges would not persist.
  }, []);

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
        reframeAfterPaint(() => rfInstance.current?.fitView({ padding: 0.2, duration: 300 }));
      });
  }, [refetchTopology, refetchDevices, refetchNeighbors]);

  const loading = topologyLoading || devicesLoading;
  // A failed fetch used to fall through to the "no topology data" branch, so a
  // daemon that could not be reached looked exactly like a network with no
  // devices in it. The comment on handleRefresh claimed these errors surfaced
  // through the per-hook error states; the page never read them.
  const loadError = topologyError ?? devicesError;

  return (
    <div className="stack-xl">
      {/* Header Card */}
      <Card className="relative z-20 border-surface-border bg-bg-surface/70">
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
                <Tooltip text={t('topology.header.graphTabTitle')}>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={view === 'graph'}
                    onClick={() => setView('graph')}
                    className={`flex items-center gap-1.5 rounded px-3 py-compact text-xs font-medium transition-colors ${
                      view === 'graph'
                        ? 'bg-status-info/20 text-status-info-strong'
                        : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
                    }`}
                  >
                    <Network className="w-3.5 h-3.5" />
                    {t('topology.header.graphTabLabel')}
                  </button>
                </Tooltip>
                <Tooltip text={t('topology.header.neighborsTabTitle')}>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={view === 'neighbors'}
                    onClick={() => setView('neighbors')}
                    className={`flex items-center gap-1.5 rounded px-3 py-compact text-xs font-medium transition-colors ${
                      view === 'neighbors'
                        ? 'bg-status-info/20 text-status-info-strong'
                        : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
                    }`}
                  >
                    <Radar className="w-3.5 h-3.5" />
                    {t('topology.header.neighborsTabLabel')}
                  </button>
                </Tooltip>
              </div>
              <Button
                variant="primary"
                size="sm"
                onClick={handleRefresh}
                icon={<RefreshCw className="w-4 h-4" />}
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
                  <ActionButton
                    variant="ghost"
                    size="sm"
                    onClick={handleResetLayout}
                    disabled={nodes.length === 0}
                    title={t('topology.header.resetButtonTitle')}
                    data-testid="topology-reset-layout-button"
                  >
                    {t('topology.header.resetButtonLabel')}
                  </ActionButton>
                  <ActionButton
                    variant={showLabels ? 'secondary' : 'ghost'}
                    size="sm"
                    onClick={() => setShowLabels(!showLabels)}
                    title={t('topology.header.labelsButtonTitle')}
                  >
                    {t('topology.header.labelsButtonLabel')}
                  </ActionButton>
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
                        <Tooltip text={entry.description} key={entry.mode}>
                          <button
                            type="button"
                            aria-pressed={active}
                            aria-label={t('topology.header.layoutModeAriaLabel', {
                              label: entry.label,
                            })}
                            onClick={() => handleLayoutModeChange(entry.mode)}
                            className={`rounded px-2.5 py-compact text-xs font-medium transition-colors ${
                              active
                                ? 'bg-status-info/20 text-status-info-strong'
                                : 'text-text-muted hover:bg-surface-hover hover:text-text-primary'
                            }`}
                          >
                            {entry.label}
                          </button>
                        </Tooltip>
                      );
                    })}
                  </div>
                  {/* Unobtrusive reminder that dragged positions live in
                      this browser's localStorage only — no backend
                      route, so a different browser/profile starts from
                      the default layout. Hidden below `lg` to keep the
                      toolbar from wrapping on narrow viewports. */}
                  <Tooltip text={t('topology.header.layoutPersistenceHint')}>
                    <button
                      type="button"
                      className="hidden lg:inline text-xs text-text-muted rounded focus-visible:outline-2 focus-visible:outline-brand-accent"
                      data-testid="topology-layout-persistence-note"
                    >
                      {t('topology.header.layoutPersistenceNote')}
                    </button>
                  </Tooltip>
                </>
              )}
            </div>
          </div>

          {/* Second header row: search + type-filter chips. Only shown
              for the graph view — the Neighbors table has its own
              filtering UI. Stays inside the header card so the toolbar
              groups visually with the title/buttons row above. */}
          {view === 'graph' && (
            <div className="mt-heading flex flex-col gap-compact sm:flex-row sm:items-center">
              <input
                type="search"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={t('topology.header.searchPlaceholder')}
                aria-label={t('topology.header.searchAriaLabel')}
                className="w-full sm:w-64 rounded-md border border-surface-border bg-bg-base/40 px-3 py-compact-md text-xs text-text-primary placeholder:text-text-muted focus:border-status-info/40 focus:outline-none focus:ring-1 focus:ring-status-info/30"
              />
              {availableSites.length > 1 && (
                <select
                  value={activeSite ?? ''}
                  onChange={(e) => handleSiteChange(e.target.value || null)}
                  aria-label={t('topology.header.siteFilterAriaLabel')}
                  data-testid="topology-site-filter"
                  className="w-full sm:w-auto rounded-md border border-surface-border bg-bg-base/40 px-3 py-compact-md text-xs text-text-primary focus:border-status-info/40 focus:outline-none focus:ring-1 focus:ring-status-info/30"
                >
                  <option value="">{t('topology.header.allSites')}</option>
                  {availableSites.map((site) => (
                    <option key={site} value={site}>
                      {site}
                    </option>
                  ))}
                </select>
              )}
              {availableTypes.length > 0 && (
                <div className="flex flex-wrap items-center gap-1.5">
                  <SmallText className="text-text-muted">
                    {t('topology.header.typesLabel')}
                  </SmallText>
                  {availableTypes.map((type) => {
                    const active = activeTypes.has(type);
                    return (
                      <Tooltip
                        text={
                          active
                            ? t('topology.header.typeFilterHideTitle', { type })
                            : t('topology.header.typeFilterShowTitle', { type })
                        }
                        key={type}
                      >
                        <button
                          type="button"
                          onClick={() => toggleType(type)}
                          aria-pressed={active}
                          className={`rounded-full border px-2.5 py-0.5 text-[11px] font-medium transition-colors ${
                            active
                              ? 'border-status-info/40 bg-status-info/20 text-status-info-strong'
                              : 'border-surface-border bg-bg-base/40 text-text-muted hover:bg-surface-hover hover:text-text-primary'
                          }`}
                        >
                          {deviceTypeLabels[normalizeDeviceType(type)]}
                        </button>
                      </Tooltip>
                    );
                  })}
                  {activeTypes.size > 0 && (
                    <Tooltip text={t('topology.header.clearTypeFiltersTitle')}>
                      <button
                        type="button"
                        onClick={() => setActiveTypes(new Set())}
                        className="rounded-full px-cell py-0.5 text-[11px] font-medium text-text-muted hover:text-text-primary underline-offset-2 hover:underline"
                      >
                        {tCommon('buttons.clear')}
                      </button>
                    </Tooltip>
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
        <Card className="relative z-0 border-surface-border bg-bg-surface/70 overflow-hidden">
          <div className="h-[calc(100vh-260px)] min-h-[520px] relative">
            {loading || loadError || nodes.length === 0 ? (
              <CanvasState loading={loading} error={loadError} onRetry={handleRefresh} />
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
                  onlyRenderVisibleElements
                  fitViewOptions={{ padding: 0.2 }}
                  minZoom={0.2}
                  maxZoom={2}
                  defaultEdgeOptions={{
                    type: 'smoothstep',
                  }}
                  proOptions={{ hideAttribution: true }}
                >
                  <TierBands tiers={tiers} left={tierExtent.left} width={tierExtent.width} />
                  <Background
                    variant={BackgroundVariant.Dots}
                    gap={20}
                    size={1}
                    color="color-mix(in srgb, var(--color-text-primary) 12%, transparent)"
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
}

export default TopologyPage;
