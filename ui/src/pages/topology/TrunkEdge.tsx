import { BaseEdge, EdgeLabelRenderer, type EdgeProps, getSmoothStepPath } from '@xyflow/react';
import type { FC } from 'react';
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
 * unobscured even for short edges. Per-side labels sit ~25 % in from
 * each end so they hug their device without overlapping the arrow.
 *
 * data.showLabels=false hides every label (just shows the line); the
 * topology header has a toggle for the user to flip.
 */

const SIDE_OFFSET_PCT = 0.22; // fraction along the path for side labels

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

  // Where the per-side labels sit. Linear interpolation along the
  // straight source→target vector — close enough for smoothstep paths
  // and avoids computing path arc length.
  const leftX = sourceX + (targetX - sourceX) * SIDE_OFFSET_PCT;
  const leftY = sourceY + (targetY - sourceY) * SIDE_OFFSET_PCT;
  const rightX = sourceX + (targetX - sourceX) * (1 - SIDE_OFFSET_PCT);
  const rightY = sourceY + (targetY - sourceY) * (1 - SIDE_OFFSET_PCT);

  const middleParts: string[] = [];
  if (linkData.vlans && linkData.vlans.length > 0) {
    middleParts.push(formatVlans(linkData.vlans));
  }
  if (linkData.speed) {
    middleParts.push(formatSpeed(linkData.speed));
  }
  const middleLabel = middleParts.join(' · ');

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={style}
        markerEnd={markerEnd}
        markerStart={markerStart}
      />
      {showLabels && (
        <EdgeLabelRenderer>
          {linkData.sourceInterface && (
            <EndLabel x={leftX} y={leftY} text={linkData.sourceInterface} />
          )}
          {middleLabel && <MiddleLabel x={labelX} y={labelY} text={middleLabel} />}
          {linkData.targetInterface && (
            <EndLabel x={rightX} y={rightY} text={linkData.targetInterface} />
          )}
        </EdgeLabelRenderer>
      )}
    </>
  );
};

const labelBoxStyle =
  'absolute pointer-events-none px-1.5 py-0.5 rounded text-[10px] font-medium ' +
  'border border-white/10 bg-gray-950/90 text-gray-200 shadow-sm whitespace-nowrap';

const EndLabel: FC<{ x: number; y: number; text: string }> = ({ x, y, text }) => (
  <div
    className={labelBoxStyle}
    style={{ transform: `translate(-50%, -50%) translate(${x}px, ${y}px)` }}
  >
    {text}
  </div>
);

const MiddleLabel: FC<{ x: number; y: number; text: string }> = ({ x, y, text }) => (
  <div
    className={`${labelBoxStyle} text-violet-200`}
    style={{ transform: `translate(-50%, -50%) translate(${x}px, ${y}px)` }}
  >
    {text}
  </div>
);

function formatVlans(vlans: number[]): string {
  if (vlans.length === 0) return '';
  if (vlans.length === 1) return `VLAN ${vlans[0]}`;
  // Compact contiguous runs: [1,2,3,5,6] → "1-3,5-6"
  const sorted = [...vlans].sort((a, b) => a - b);
  const runs: string[] = [];
  let start = sorted[0];
  let prev = sorted[0];
  for (let i = 1; i < sorted.length; i++) {
    const v = sorted[i];
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
