import { DebugLevelControl } from '../../components/debug/DebugLevelControl';
import { Card } from '../../shared/Card';

/**
 * GlobalDebugLevelCard — Runtime-page home for the global debug-level control.
 * The control itself lives in DebugLevelControl so the Debug Console can mount
 * the same surface.
 */
export function GlobalDebugLevelCard() {
  return (
    <Card>
      <div className="stack">
        <DebugLevelControl />
      </div>
    </Card>
  );
}
