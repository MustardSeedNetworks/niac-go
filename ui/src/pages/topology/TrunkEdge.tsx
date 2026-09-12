import { BaseEdge, EdgeLabelRenderer, type EdgeProps, getSmoothStepPath } from '@xyflow/react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { NODE_HEIGHT, NODE_WIDTH } from './layout';
import type { LinkEdgeData } from './types';

/**
 * TrunkEdge is the custom ReactFlow edge used for every topology
 * link. Renders the line and arrows via BaseEdge, then layers three
 * floating labels via EdgeLabelRenderer:
 *
 *   - left near the source: the source-side interface name (e.g. Gi0/1)
 *   - centre on the line:   the shared metadata (VLANs, speed)
 *   - right near the target: the target-side interface name
 *
 * Splitting like this keeps the centre of the line — and the arrows —
 * unobscured even for short edges. Per-side labels sit ~22 % in from
 * each end so they hug their device without overlapping the arrow.
 *
 * data.showLabels=false hides every label (just shows the line); the
 * topology header has a toggle for the user to flip.
 *
 * data.discovered=true renders the stroke dashed so runtime
 * LLDP/CDP/EDP/FDP-inferred edges are visually distinct from
 * config-declared trunk_ports.
 *
 * data.focusOpacity (set when a node is "selected" for neighbourhood
 * highlighting) fades the edge to that opacity so non-adjacent links
 * recede to the background.
 *
 * data.hovered (driven by ReactFlow's onEdgeMouseEnter / Leave at the
 * page level) surfaces the richer tooltip with full interface names,
 * speed/duplex/status, and the complete VLAN list. Lifting hover
 * state to the page avoids putting mouse handlers on SVG paths
 * directly (a11y rule), and ReactFlow's built-in interactionWidth
 * provides an invisible wider hit area for easy targeting.
 */

/**
 * Fallback fraction along the path for the side labels.
 *
 * `createEdges` normally supplies a per-edge offset that separates the labels
 * of edges sharing a device; this covers an edge built without one.
 */
/**
 * Label geometry, in graph units.
 *
 * Edge labels scale with the canvas, so a label occupies the same number of
 * graph units at every zoom and these do not need a zoom term. The row height
 * is one label plus a gap: siblings stacked a row apart are separated by more
 * than their own height, which is what stops them overlapping.
 */
const LABEL_HEIGHT = 21;
const LABEL_ROW = LABEL_HEIGHT + 8;

/**
 * How far the first label sits from its device.
 *
 * Labels are placed along the straight line between endpoints while the edge
 * itself is a smoothstep path, so near a device the two diverge. Starting
 * outside the node's own footprint — its half-diagonal, plus half a label —
 * keeps a label off the device it belongs to, which a shorter start did not:
 * an arriving edge's label sat on the node it was arriving at.
 */
const LABEL_START = Math.hypot(NODE_WIDTH, NODE_HEIGHT) / 2 + LABEL_HEIGHT;

/** The edge path's own length, or the straight-line fallback. */
function pathLength(path: string, fallback: number): number {
  const element = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  element.setAttribute('d', path);
  if (typeof element.getTotalLength !== 'function') {
    return fallback;
  }
  const total = element.getTotalLength();
  return total === 0 ? fallback : total;
}

/**
 * A point the given distance along the edge's own path.
 *
 * Interpolating the straight line between the endpoints instead puts labels
 * beside the line rather than on it, and near a device the two diverge enough
 * that a label lands on the node — which is where the arriving edges' labels
 * ended up. Measuring the real path costs one detached element per call and
 * is exact.
 *
 * `fallback` is the straight-line point, used where the platform has no path
 * measurement (jsdom in unit tests) rather than returning nothing.
 */
function pointAlong(
  path: string,
  distance: number,
  fallback: { x: number; y: number },
): { x: number; y: number } {
  const element = document.createElementNS('http://www.w3.org/2000/svg', 'path');
  element.setAttribute('d', path);
  if (typeof element.getTotalLength !== 'function') {
    return fallback;
  }
  const total = element.getTotalLength();
  if (total === 0) {
    return fallback;
  }
  const point = element.getPointAtLength(Math.max(0, Math.min(distance, total)));
  return { x: point.x, y: point.y };
}

/**
 * Graph units a label box occupies, from the 10px font plus the box's padding.
 *
 * The floor matters: for short text like "1M" the padding and border dominate
 * the width, and estimating from the character count alone let an interface
 * label be placed on top of the line's own speed label.
 */
const LABEL_MIN_LENGTH = 48;

function labelLength(text: string): number {
  return Math.max(text.length * 5.6 + 14, LABEL_MIN_LENGTH);
}

export const TrunkEdge: FC<EdgeProps> = ({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style,
  markerEnd,
  markerStart,
  data,
}) => {
  const linkData = (data ?? {}) as LinkEdgeData;
  const [edgePath, labelX, labelY] = getSmoothStepPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const showLabels = linkData.showLabels !== false; // default on

  const focusOpacity = linkData.focusOpacity ?? 1;
  const hovered = linkData.hovered === true;

  // Dashed stroke for discovered edges; otherwise inherit whatever
  // the caller's style.strokeDasharray says (typically undefined → solid).
  const finalStyle: React.CSSProperties = {
    ...style,
    opacity: focusOpacity,
    ...(linkData.discovered ? { strokeDasharray: '6 4' } : {}),
  };

  const middleParts: string[] = [];
  if (linkData.vlans && linkData.vlans.length > 0) {
    middleParts.push(formatVlans(linkData.vlans));
  }
  // Guard the formatted value, not the raw field: '0' is truthy but
  // formatSpeed returns '', which would still be joined in and leave the
  // label reading 'VLAN 10 · '.
  const middleSpeed = linkData.speed ? formatSpeed(linkData.speed) : '';
  if (middleSpeed) {
    middleParts.push(middleSpeed);
  }
  const middleLabel = middleParts.join(' · ');

  // Where the per-side labels sit: a fixed distance along the edge from their
  // own device, stepped one row per sibling.
  //
  // Distances rather than fractions, because what a label must clear is
  // another label, and that is a length — a fraction of a long edge and a
  // fraction of a short one are different amounts of room. An edge alone
  // cannot know how many siblings crowd its device, so createEdges supplies
  // the index and the edge turns it into a position.
  const edgeLength = pathLength(edgePath, Math.hypot(targetX - sourceX, targetY - sourceY));
  const sourceAlong = LABEL_START + (linkData.sourceSiblingIndex ?? 0) * LABEL_ROW;
  const targetAlong = LABEL_START + (linkData.targetSiblingIndex ?? 0) * LABEL_ROW;

  const at = (distance: number): { x: number; y: number } =>
    pointAlong(edgePath, distance, {
      x: sourceX + (targetX - sourceX) * (edgeLength === 0 ? 0 : distance / edgeLength),
      y: sourceY + (targetY - sourceY) * (edgeLength === 0 ? 0 : distance / edgeLength),
    });
  const left = at(sourceAlong);
  const right = at(edgeLength - targetAlong);
  // The middle label is placed at the path's midpoint by length, in the same
  // measure as the end labels, rather than at ReactFlow's labelX/labelY — that
  // is the path's geometric centre, which on a smoothstep run sits somewhere
  // else entirely, so reserving room around it as if it were the midpoint let
  // an end label land on top of it.
  const middle = at(edgeLength / 2);

  // A label is drawn when its slot fits: its own stacked position, plus the
  // text it has to hold, has to stay clear of the line's middle label and of
  // the label stacked at the other end. Where it does not fit there is no
  // arrangement that would show it legibly, so it is left out rather than
  // drawn over its neighbour.
  const middleNeeds = labelLength(middleLabel);
  const roomForEnds = (edgeLength - middleNeeds) / 2;
  const sourceFits = sourceAlong + labelLength(linkData.sourceInterface ?? '') / 2 <= roomForEnds;
  const targetFits = targetAlong + labelLength(linkData.targetInterface ?? '') / 2 <= roomForEnds;

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={finalStyle}
        markerEnd={markerEnd}
        markerStart={markerStart}
      />
      {showLabels && (
        <EdgeLabelRenderer>
          {sourceFits && linkData.sourceInterface && (
            <EndLabel
              x={left.x}
              y={left.y}
              text={linkData.sourceInterface}
              opacity={focusOpacity}
            />
          )}
          {middleLabel && (
            <MiddleLabel x={middle.x} y={middle.y} text={middleLabel} opacity={focusOpacity} />
          )}
          {targetFits && linkData.targetInterface && (
            <EndLabel
              x={right.x}
              y={right.y}
              text={linkData.targetInterface}
              opacity={focusOpacity}
            />
          )}
        </EdgeLabelRenderer>
      )}
      {hovered && (
        <EdgeLabelRenderer>
          <EdgeTooltip x={labelX} y={labelY} data={linkData} />
        </EdgeLabelRenderer>
      )}
    </>
  );
};

const labelBoxStyle =
  'absolute pointer-events-none px-1.5 py-0.5 rounded text-[10px] font-medium ' +
  'border border-surface-border bg-bg-base/90 text-text-primary shadow-sm whitespace-nowrap';

const EndLabel: FC<{ x: number; y: number; text: string; opacity: number }> = ({
  x,
  y,
  text,
  opacity,
}) => (
  <div
    className={labelBoxStyle}
    style={{ transform: `translate(-50%, -50%) translate(${x}px, ${y}px)`, opacity }}
  >
    {text}
  </div>
);

const MiddleLabel: FC<{ x: number; y: number; text: string; opacity: number }> = ({
  x,
  y,
  text,
  opacity,
}) => (
  <div
    className={`${labelBoxStyle} text-brand-accent`}
    style={{ transform: `translate(-50%, -50%) translate(${x}px, ${y}px)`, opacity }}
  >
    {text}
  </div>
);

const EdgeTooltip: FC<{ x: number; y: number; data: LinkEdgeData }> = ({ x, y, data }) => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const rows: [string, string][] = [];
  if (data.sourceInterface || data.targetInterface) {
    rows.push([
      tCommon('labels.interfaces'),
      `${data.sourceInterface ?? '?'} ↔ ${data.targetInterface ?? '?'}`,
    ]);
  }
  if (data.vlans && data.vlans.length > 0) {
    rows.push([t('topology.trunkEdge.rowVlans'), data.vlans.join(', ')]);
  }
  const speedLabel = data.speed ? formatSpeed(data.speed) : '';
  if (speedLabel) {
    rows.push([t('topology.trunkEdge.rowSpeed'), speedLabel]);
  }
  if (data.duplex) {
    rows.push([t('topology.trunkEdge.rowDuplex'), data.duplex]);
  }
  if (data.status) {
    rows.push([tCommon('labels.status'), data.status]);
  }
  if (data.linkType) {
    rows.push([tCommon('labels.type'), data.linkType]);
  }
  if (typeof data.utilizationPercent === 'number' && data.utilizationPercent > 0) {
    rows.push([t('topology.trunkEdge.rowUtilisation'), `${data.utilizationPercent.toFixed(0)} %`]);
  }
  rows.push([
    t('topology.trunkEdge.rowSource'),
    data.discovered
      ? t('topology.trunkEdge.sourceDiscovered')
      : t('topology.trunkEdge.sourceDeclared'),
  ]);

  return (
    <div
      className="absolute pointer-events-none rounded-lg border border-surface-border bg-bg-base/95 px-3 py-row text-xs text-text-primary shadow-lg z-50"
      style={{ transform: `translate(-50%, calc(-100% - 12px)) translate(${x}px, ${y}px)` }}
    >
      <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
        {rows.map(([k, v]) => (
          <div key={k} className="contents">
            <span className="text-text-muted">{k}</span>
            <span className="font-medium text-text-primary">{v}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

function formatVlans(vlans: number[]): string {
  const [firstVlan, ...restVlans] = [...vlans].sort((a, b) => a - b);
  if (firstVlan === undefined) {
    return '';
  }
  if (restVlans.length === 0) {
    return `VLAN ${firstVlan}`;
  }

  const runs: string[] = [];
  let start = firstVlan;
  let prev = firstVlan;
  for (const v of restVlans) {
    if (v === prev + 1) {
      prev = v;
      continue;
    }
    runs.push(start === prev ? String(start) : `${start}-${prev}`);
    start = v;
    prev = v;
  }
  runs.push(start === prev ? String(start) : `${start}-${prev}`);
  return `VLANs ${runs.join(',')}`;
}

function formatSpeed(speedMbps: string): string {
  const n = Number.parseInt(speedMbps, 10);
  if (!Number.isFinite(n) || n <= 0) return '';
  if (n >= 1000) return `${n / 1000}G`;
  return `${n}M`;
}
