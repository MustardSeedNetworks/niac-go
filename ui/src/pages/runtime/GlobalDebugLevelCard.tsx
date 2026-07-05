import type { FC } from 'react';
import { DebugLevelControl } from '../../components/debug/DebugLevelControl';
import { Card, CardContent } from '../../ui/Card';

/**
 * GlobalDebugLevelCard — Runtime-page home for the global debug-level control.
 * The control itself lives in DebugLevelControl so the Debug Console can mount
 * the same surface.
 */
export const GlobalDebugLevelCard: FC = () => (
  <Card className="border-surface-border bg-bg-surface/70">
    <CardContent className="stack">
      <DebugLevelControl />
    </CardContent>
  </Card>
);
