/**
 * Inspector — the Inspector page archetype: a filter bar over a record list,
 * with the selected record's structure and payload beside it.
 *
 * Shared shell pattern, kept visually and behaviourally consistent across
 * seed / stem / niac / trellis by convention; each repo owns this file
 * independently (no master, no sync). All colours reference theme tokens.
 *
 * Presentational only — selection, filtering and data live in the page, and
 * the record list itself stays page-owned because its markup legitimately
 * differs per record type (the live capture renders a button list, the pcap
 * analyzer a table). What this file fixes is the frame around them, which had
 * drifted into mirror images: the same archetype, on two tabs of one route,
 * gave the record list 3/12 of the width in one and 8/12 in the other.
 *
 * Where a record's colour comes from is the one rule that differs from the
 * List + detail archetype. There, colour comes from state and never category.
 * An Inspector's list is a dissection surface whose primary axis IS category —
 * protocol identity is the thing being read, and Wireshark-style coloring
 * rules are a first-class feature, not decoration. So category colour is
 * legitimate here and nowhere else. Severity-coloured lists (the debug
 * console) keep using state colour; the two coexist deliberately.
 *
 * Usage:
 *   <Inspector filter={<FilterBar … />}>
 *     <InspectorRecords><PacketList … /></InspectorRecords>
 *     <InspectorPanes>
 *       <InspectorPane label="Hex dump"><HexDumpViewer … /></InspectorPane>
 *       <InspectorPane label="Packet details" scroll><PacketDetails … /></InspectorPane>
 *     </InspectorPanes>
 *   </Inspector>
 */
import type { FC, ReactNode } from 'react';
import { cn } from '../styles/theme';
import { Card, CardContent } from './Card';

/**
 * Height of the record list and of the pane column beside it. They match so
 * the two columns bottom-align; a short pane column next to a tall list reads
 * as a rendering fault rather than as a design.
 */
const COLUMN_HEIGHT = 'h-[600px]';

export const Inspector: FC<{ filter?: ReactNode; children: ReactNode }> = ({
  filter,
  children,
}) => (
  <div className="stack-xl">
    {filter ? <div>{filter}</div> : null}
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-spacious">{children}</div>
  </div>
);

/**
 * The record list column. Deliberately the wider half: the list is what the
 * operator reads and scans, while the panes answer a question about the one
 * row already chosen.
 */
export const InspectorRecords: FC<{ children: ReactNode }> = ({ children }) => (
  <div className="lg:col-span-7 xl:col-span-8">
    <div className={COLUMN_HEIGHT}>{children}</div>
  </div>
);

export const InspectorPanes: FC<{ children: ReactNode }> = ({ children }) => (
  <div className={cn('lg:col-span-5 xl:col-span-4 flex flex-col gap-spacious', COLUMN_HEIGHT)}>
    {children}
  </div>
);

interface InspectorPaneProps {
  /** Kicker above the pane. Prose, so it is not monospaced. */
  label: string;
  /** Content that scrolls rather than clipping — a structure tree, usually. */
  scroll?: boolean;
  children: ReactNode;
}

/**
 * One labelled pane. Panes share the column height evenly instead of carrying
 * hand-set pixel heights, which is what let the two implementations drift to
 * 350/220 and 280/280 for the same two panes.
 */
export const InspectorPane: FC<InspectorPaneProps> = ({ label, scroll = false, children }) => (
  <Card className="border-surface-border bg-bg-surface/70 flex-1 min-h-0">
    <CardContent className="h-full flex flex-col">
      <p className="text-xs uppercase tracking-wide font-semibold text-text-muted mb-heading">
        {label}
      </p>
      <div className={cn('flex-1 min-h-0', scroll && 'overflow-y-auto')}>{children}</div>
    </CardContent>
  </Card>
);
