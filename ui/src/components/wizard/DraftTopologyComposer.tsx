import {
  Background,
  BackgroundVariant,
  type Connection,
  type EdgeTypes,
  type NodeTypes,
  ReactFlow,
  useEdgesState,
  useNodesState,
} from '@xyflow/react';
import { Cable, Plus } from 'lucide-react';
import { type FC, useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  type DraftTopologyEndpoint,
  type DraftTopologyLinkProperties,
  type DraftTopologyMutation,
  mutateScenarioDraftTopology,
  type ScenarioDraft,
} from '../../api/library-client';
import { fetchScenarioProfiles } from '../../api/scenario-client';
import type { TopologyLink } from '../../api/types';
import { iconSizes } from '../../constants/sizes';
import { useApiResource } from '../../hooks/useApiResource';
import { useErrorToast } from '../../hooks/useErrorToast';
import {
  createEdges,
  DeviceNode,
  type DeviceNodeType,
  type LinkEdge,
  layoutNodes,
} from '../../pages/topology';
import { TrunkEdge } from '../../pages/topology/TrunkEdge';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';
import {
  DeviceEditorModal,
  type DeviceEditorState,
  LinkEditorModal,
  type LinkEditorState,
} from './DraftTopologyModals';
import { parseDraftTopology } from './draft-topology';

const nodeTypes: NodeTypes = { device: DeviceNode };
const edgeTypes: EdgeTypes = { trunk: TrunkEdge };

interface DraftTopologyComposerProps {
  draft: ScenarioDraft;
  onDraftUpdate: (draft: ScenarioDraft) => void;
  onBusyChange: (busy: boolean) => void;
}

const emptyDevice: DeviceEditorState = {
  name: '',
  role: 'access',
  interfaceCount: '4',
  speed: '1000',
};

function deterministicMACSuffix(name: string) {
  let hash = 2166136261;
  for (const character of name) {
    hash ^= character.charCodeAt(0);
    hash = Math.imul(hash, 16777619);
  }
  return hash & 0xffffff || 1;
}

function parseVLANs(value: string): number[] | null {
  if (!value.trim()) return [];
  const parts = value.split(',').map((part) => part.trim());
  if (parts.some((part) => !/^\d+$/.test(part))) return null;
  const vlans = parts.map(Number);
  return [...new Set(vlans)];
}

function linkState(link: TopologyLink): LinkEditorState {
  return {
    source: link.source,
    target: link.target,
    sourceInterface: link.sourceInterface ?? '',
    targetInterface: link.targetInterface ?? '',
    vlans: (link.vlans ?? []).join(', '),
    nativeVlan: link.nativeVlan ? String(link.nativeVlan) : '',
    fdbOnly: link.fdbOnly ?? false,
    existing: true,
  };
}

export const DraftTopologyComposer: FC<DraftTopologyComposerProps> = ({
  draft,
  onDraftUpdate,
  onBusyChange,
}) => {
  const { t } = useTranslation('pages');
  const showError = useErrorToast();
  const {
    data: profiles,
    error: profilesError,
    loading: profilesLoading,
    refetch: refetchProfiles,
  } = useApiResource(fetchScenarioProfiles, []);
  const [busy, setBusy] = useState(false);
  const [deviceEditor, setDeviceEditor] = useState<DeviceEditorState | null>(null);
  const [linkEditor, setLinkEditor] = useState<LinkEditorState | null>(null);
  const model = useMemo(() => parseDraftTopology(draft.content), [draft.content]);
  const layoutedNodes = useMemo(() => {
    const result = layoutNodes(model.devices, model.links);
    return result.map((node) => ({
      ...node,
      position: model.positions[node.id] ?? node.position,
    }));
  }, [model]);
  const layoutedEdges = useMemo(
    () =>
      createEdges(model.links).map((edge, index) => ({
        ...edge,
        selectable: model.links[index]?.reciprocal !== false,
      })),
    [model.links],
  );
  const [nodes, setNodes, onNodesChange] = useNodesState<DeviceNodeType>(layoutedNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState<LinkEdge>(layoutedEdges);

  useEffect(() => setNodes(layoutedNodes), [layoutedNodes, setNodes]);
  useEffect(() => setEdges(layoutedEdges), [layoutedEdges, setEdges]);

  const applyMutation = useCallback(
    async (mutation: DraftTopologyMutation) => {
      setBusy(true);
      onBusyChange(true);
      try {
        const updated = await mutateScenarioDraftTopology(draft.name, draft.revision, mutation);
        onDraftUpdate(updated);
        return true;
      } catch (error) {
        showError(error);
        setNodes(layoutedNodes);
        return false;
      } finally {
        setBusy(false);
        onBusyChange(false);
      }
    },
    [draft.name, draft.revision, layoutedNodes, onBusyChange, onDraftUpdate, setNodes, showError],
  );

  const openConnection = useCallback(
    (connection: Connection) => {
      if (!(connection.source && connection.target) || connection.source === connection.target)
        return;
      if (
        model.segmented &&
        model.segmentByDevice[connection.source] !== model.segmentByDevice[connection.target]
      ) {
        showError(new Error(t('newSimWizard.topology.crossSegmentError')));
        return;
      }
      setLinkEditor({
        source: connection.source,
        target: connection.target,
        sourceInterface: '',
        targetInterface: '',
        vlans: '',
        nativeVlan: '',
        fdbOnly: false,
        existing: false,
      });
    },
    [model.segmentByDevice, model.segmented, showError, t],
  );

  const interfaceOptions = useCallback(
    (device: string, selected: string) =>
      (model.interfaces[device] ?? [])
        .filter(
          (iface) =>
            (!iface.occupied || iface.name === selected) &&
            ['ethernet', 'other', ''].includes(iface.type),
        )
        .map((iface) => ({
          value: iface.name,
          label: `${iface.name}${iface.speed ? ` · ${iface.speed} Mbps` : ''}`,
        })),
    [model.interfaces],
  );

  const saveLink = useCallback(async () => {
    if (!linkEditor) return;
    const vlans = parseVLANs(linkEditor.vlans);
    if (!vlans) return;
    const nativeVlan = linkEditor.nativeVlan ? Number.parseInt(linkEditor.nativeVlan, 10) : 0;
    const properties: DraftTopologyLinkProperties = {
      vlans,
      nativeVlan,
      fdbOnly: linkEditor.fdbOnly,
    };
    const source: DraftTopologyEndpoint = {
      device: linkEditor.source,
      interface: linkEditor.sourceInterface,
    };
    const target: DraftTopologyEndpoint = {
      device: linkEditor.target,
      interface: linkEditor.targetInterface,
    };
    if (
      await applyMutation({
        operation: linkEditor.existing ? 'update_link' : 'connect',
        link: { source, target, properties },
      })
    ) {
      setLinkEditor(null);
    }
  }, [applyMutation, linkEditor]);

  const disconnect = useCallback(async () => {
    if (!linkEditor?.existing) return;
    const source = { device: linkEditor.source, interface: linkEditor.sourceInterface };
    const target = { device: linkEditor.target, interface: linkEditor.targetInterface };
    if (await applyMutation({ operation: 'disconnect', link: { source, target } })) {
      setLinkEditor(null);
    }
  }, [applyMutation, linkEditor]);

  const addDevice = useCallback(async () => {
    if (!deviceEditor || !profiles) return;
    const profile = profiles.find((item) => item.role === deviceEditor.role);
    if (!profile) return;
    const interfaceCount = Number.parseInt(deviceEditor.interfaceCount, 10);
    const speed = Number.parseInt(deviceEditor.speed, 10);
    const interfaces = profile.interfaces?.length
      ? profile.interfaces.map((iface) => ({
          name: iface.name,
          type: iface.type,
          mtu: iface.mtu ?? 1500,
          speed: Math.round((iface.speed ?? 1_000_000_000) / 1_000_000),
          duplex: 'full' as const,
          adminStatus: iface.adminStatus === 'down' ? ('down' as const) : ('up' as const),
          operStatus: iface.operStatus === 'down' ? ('down' as const) : ('up' as const),
        }))
      : Array.from({ length: interfaceCount }, (_, index) => ({
          name: `Ethernet1/${index + 1}`,
          type: 'ethernet',
          mtu: speed >= 10000 ? 9000 : 1500,
          speed,
          duplex: 'full' as const,
          adminStatus: 'up' as const,
          operStatus: 'up' as const,
        }));
    if (
      await applyMutation({
        operation: 'add_device',
        device: {
          name: deviceEditor.name.trim(),
          type: profile.deviceType,
          vendor: profile.vendor,
          macSuffix: deterministicMACSuffix(deviceEditor.name.trim()),
          sysObjectId: profile.sysObjectId,
          profileRole: profile.role,
          interfaces,
          properties: {
            role: profile.role,
            model: profile.model,
            platform: profile.platform,
            software: profile.software,
          },
        },
      })
    ) {
      setDeviceEditor(null);
    }
  }, [applyMutation, deviceEditor, profiles]);

  const linkValid = useMemo(() => {
    if (!linkEditor?.sourceInterface || !linkEditor.targetInterface) return false;
    const vlans = parseVLANs(linkEditor.vlans);
    if (!vlans || vlans.some((vlan) => vlan < 1 || vlan > 4094)) return false;
    if (!linkEditor.nativeVlan) return true;
    return vlans.includes(Number.parseInt(linkEditor.nativeVlan, 10));
  }, [linkEditor]);

  const deviceValid = useMemo(() => {
    if (!deviceEditor || !profiles) return false;
    const count = Number.parseInt(deviceEditor.interfaceCount, 10);
    const speed = Number.parseInt(deviceEditor.speed, 10);
    const selectedProfile = profiles.find((profile) => profile.role === deviceEditor.role);
    const maximumInterfaces = selectedProfile?.interfaces?.length ? 4096 : 64;
    return (
      /^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$/.test(deviceEditor.name.trim()) &&
      !model.devices.some((device) => device.name === deviceEditor.name.trim()) &&
      profiles.some((profile) => profile.role === deviceEditor.role) &&
      count >= 1 &&
      count <= maximumInterfaces &&
      speed >= 10 &&
      speed <= 400000
    );
  }, [deviceEditor, model.devices, profiles]);

  return (
    <div className="stack">
      <div className="flex flex-wrap items-center justify-between gap-default">
        <SmallText className="text-text-muted">{t('newSimWizard.topology.help')}</SmallText>
        <Button
          tone="violet"
          leftIcon={<Plus className={iconSizes.md} />}
          disabled={busy || !profiles || model.segmented}
          onClick={() => setDeviceEditor(emptyDevice)}
        >
          {t('newSimWizard.topology.addDevice')}
        </Button>
      </div>
      {profilesError && (
        <div className="flex items-center justify-between gap-default rounded-lg border border-status-error/40 bg-status-error/10 px-3 py-2">
          <SmallText className="text-status-error">
            {t('newSimWizard.topology.profileLoadError')}
          </SmallText>
          <Button
            size="sm"
            variant="outline"
            tone="red"
            loading={profilesLoading}
            onClick={() => void refetchProfiles()}
          >
            {t('newSimWizard.topology.retry')}
          </Button>
        </div>
      )}
      <div className="h-[36rem] overflow-hidden rounded-xl border border-surface-border bg-bg-base/50">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={openConnection}
          onNodeDragStop={(_, node) =>
            void applyMutation({
              operation: 'move_device',
              position: { device: node.id, x: node.position.x, y: node.position.y },
            })
          }
          onEdgeClick={(_, edge) => {
            const link = model.links.find(
              (item) =>
                item.source === edge.source &&
                item.target === edge.target &&
                item.sourceInterface === edge.data?.sourceInterface &&
                item.targetInterface === edge.data?.targetInterface,
            );
            if (link && link.reciprocal !== false) setLinkEditor(linkState(link));
          }}
          nodesConnectable={!busy}
          nodesDraggable={!busy}
          elementsSelectable={!busy}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView={true}
          minZoom={0.1}
          maxZoom={2}
          proOptions={{ hideAttribution: true }}
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={20}
            size={1}
            color="var(--color-surface-border)"
          />
        </ReactFlow>
      </div>
      <SmallText className="flex items-center gap-inline text-text-muted">
        <Cable className={iconSizes.sm} /> {t('newSimWizard.topology.connectHint')}
      </SmallText>
      <DeviceEditorModal
        state={deviceEditor}
        profiles={profiles ?? []}
        valid={deviceValid}
        busy={busy}
        onChange={setDeviceEditor}
        onSave={() => void addDevice()}
      />
      <LinkEditorModal
        state={linkEditor}
        valid={linkValid}
        busy={busy}
        interfaceOptions={interfaceOptions}
        onChange={setLinkEditor}
        onSave={() => void saveLink()}
        onDisconnect={() => void disconnect()}
      />
    </div>
  );
};
