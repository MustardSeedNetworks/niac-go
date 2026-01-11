import { type FC, useCallback, useMemo } from 'react';
import {
  Download,
  Trash2,
  RotateCcw,
  ChevronLeft,
  ChevronRight,
  CheckCircle2,
  AlertCircle,
  FileCheck,
} from 'lucide-react';
import {
  Card,
  CardContent,
  Button,
  SmallText,
  Tag,
  H2,
} from '../../ui';
import type { MergeDecision, DiffBlock } from './DiffViewer';
import { YamlViewer } from './YamlEditor';

interface MergeControlsProps {
  diffBlocks: DiffBlock[];
  mergeDecisions: Map<string, MergeDecision>;
  onAcceptAllLeft: () => void;
  onAcceptAllRight: () => void;
  onReset: () => void;
  onExport: () => void;
  onPreview: () => void;
  disabled?: boolean;
  leftLabel?: string;
  rightLabel?: string;
}

/**
 * Merge action controls for the config diff page
 */
export const MergeControls: FC<MergeControlsProps> = ({
  diffBlocks,
  mergeDecisions,
  onAcceptAllLeft,
  onAcceptAllRight,
  onReset,
  onExport,
  onPreview,
  disabled = false,
  leftLabel = 'Original',
  rightLabel = 'Modified',
}) => {
  // Calculate statistics
  const stats = useMemo(() => {
    const changedBlocks = diffBlocks.filter((b) => b.type !== 'unchanged');
    const totalChanges = changedBlocks.length;
    const decisionsCount = mergeDecisions.size;

    let leftCount = 0;
    let rightCount = 0;
    let bothCount = 0;

    for (const decision of mergeDecisions.values()) {
      if (decision.choice === 'left') leftCount++;
      else if (decision.choice === 'right') rightCount++;
      else if (decision.choice === 'both') bothCount++;
    }

    const isComplete = decisionsCount === totalChanges;
    const progress = totalChanges > 0 ? Math.round((decisionsCount / totalChanges) * 100) : 100;

    return {
      totalChanges,
      decisionsCount,
      leftCount,
      rightCount,
      bothCount,
      isComplete,
      progress,
    };
  }, [diffBlocks, mergeDecisions]);

  const handleAcceptAllLeft = useCallback(() => {
    if (window.confirm(`Accept all changes from "${leftLabel}"? This will override current decisions.`)) {
      onAcceptAllLeft();
    }
  }, [onAcceptAllLeft, leftLabel]);

  const handleAcceptAllRight = useCallback(() => {
    if (window.confirm(`Accept all changes from "${rightLabel}"? This will override current decisions.`)) {
      onAcceptAllRight();
    }
  }, [onAcceptAllRight, rightLabel]);

  const handleReset = useCallback(() => {
    if (mergeDecisions.size > 0) {
      if (window.confirm('Reset all merge decisions? This cannot be undone.')) {
        onReset();
      }
    }
  }, [onReset, mergeDecisions.size]);

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <H2 className="mb-0 flex items-center gap-2">
            <FileCheck className="h-5 w-5 text-violet-300" />
            Merge Controls
          </H2>
          {stats.totalChanges > 0 && (
            <div className="flex items-center gap-2">
              {stats.isComplete ? (
                <Tag colorScheme="green" className="flex items-center gap-1">
                  <CheckCircle2 className="h-3 w-3" />
                  All resolved
                </Tag>
              ) : (
                <Tag colorScheme="yellow" className="flex items-center gap-1">
                  <AlertCircle className="h-3 w-3" />
                  {stats.totalChanges - stats.decisionsCount} pending
                </Tag>
              )}
            </div>
          )}
        </div>

        {/* Progress bar */}
        {stats.totalChanges > 0 && (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <SmallText className="text-gray-400">
                Resolution progress
              </SmallText>
              <SmallText className="text-gray-300">
                {stats.decisionsCount} / {stats.totalChanges} ({stats.progress}%)
              </SmallText>
            </div>
            <div className="h-2 rounded-full bg-gray-800 overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-violet-500 to-violet-400 transition-all duration-300"
                style={{ width: `${stats.progress}%` }}
              />
            </div>
          </div>
        )}

        {/* Decision breakdown */}
        {stats.decisionsCount > 0 && (
          <div className="flex items-center gap-4 flex-wrap">
            <SmallText className="text-gray-400">Decisions:</SmallText>
            {stats.leftCount > 0 && (
              <Tag colorScheme="blue" className="text-xs">
                <ChevronLeft className="h-3 w-3 mr-1" />
                {stats.leftCount} left
              </Tag>
            )}
            {stats.rightCount > 0 && (
              <Tag colorScheme="green" className="text-xs">
                <ChevronRight className="h-3 w-3 mr-1" />
                {stats.rightCount} right
              </Tag>
            )}
            {stats.bothCount > 0 && (
              <Tag colorScheme="purple" className="text-xs">
                {stats.bothCount} both
              </Tag>
            )}
          </div>
        )}

        {/* Quick actions */}
        <div className="space-y-3">
          <SmallText className="text-gray-400 uppercase tracking-wide font-semibold">
            Quick Actions
          </SmallText>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Button
              variant="outline"
              disabled={disabled || stats.totalChanges === 0}
              onClick={handleAcceptAllLeft}
              leftIcon={<ChevronLeft className="h-4 w-4" />}
              className="justify-start"
            >
              Accept all from {leftLabel}
            </Button>

            <Button
              variant="outline"
              disabled={disabled || stats.totalChanges === 0}
              onClick={handleAcceptAllRight}
              leftIcon={<ChevronRight className="h-4 w-4" />}
              className="justify-start"
            >
              Accept all from {rightLabel}
            </Button>
          </div>
        </div>

        {/* Main actions */}
        <div className="flex flex-wrap gap-3 pt-2 border-t border-white/10">
          <Button
            tone="violet"
            disabled={disabled || stats.totalChanges === 0}
            onClick={onPreview}
            leftIcon={<FileCheck className="h-4 w-4" />}
          >
            Preview Merged
          </Button>

          <Button
            tone="violet"
            disabled={disabled || !stats.isComplete}
            onClick={onExport}
            leftIcon={<Download className="h-4 w-4" />}
          >
            Export Result
          </Button>

          <Button
            variant="outline"
            disabled={disabled || stats.decisionsCount === 0}
            onClick={handleReset}
            leftIcon={<RotateCcw className="h-4 w-4" />}
          >
            Reset Decisions
          </Button>

          <Button
            variant="ghost"
            disabled={disabled}
            onClick={() => {
              if (window.confirm('Clear all files and start over?')) {
                window.location.reload();
              }
            }}
            leftIcon={<Trash2 className="h-4 w-4" />}
          >
            Clear All
          </Button>
        </div>

        {/* Help text */}
        <SmallText className="text-gray-500">
          {stats.totalChanges === 0
            ? 'Upload two YAML config files to start comparing and merging.'
            : stats.isComplete
              ? 'All changes resolved. You can now export the merged configuration.'
              : 'Click the merge buttons on each changed block to decide how to merge.'}
        </SmallText>
      </CardContent>
    </Card>
  );
};

/**
 * Preview modal for merged content
 */
interface MergePreviewModalProps {
  content: string;
  onClose: () => void;
  onExport: () => void;
}

export const MergePreviewModal: FC<MergePreviewModalProps> = ({
  content,
  onClose,
  onExport,
}) => {
  const lineCount = useMemo(() => content.split('\n').length, [content]);

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(content);
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement('textarea');
      textarea.value = content;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
    }
  }, [content]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="preview-modal-title"
    >
      <div
        className="relative mx-4 max-h-[90vh] w-full max-w-4xl overflow-hidden rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-white/10 px-6 py-4 flex-shrink-0">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-violet-500/20 p-2">
              <FileCheck className="h-5 w-5 text-violet-300" />
            </div>
            <div>
              <h2 id="preview-modal-title" className="text-lg font-semibold text-white">
                Merged Configuration Preview
              </h2>
              <SmallText className="text-gray-400">
                {lineCount} lines
              </SmallText>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 text-gray-400 hover:bg-white/10 hover:text-white transition-colors"
            aria-label="Close modal"
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Modal Content */}
        <div className="flex-1 overflow-auto p-6">
          <YamlViewer
            value={content}
            height="100%"
            minHeight="300px"
            maxHeight="60vh"
            showLineNumbers={true}
            showFoldGutter={true}
          />
        </div>

        {/* Modal Footer */}
        <div className="flex justify-end gap-3 border-t border-white/10 px-6 py-4 bg-gray-950/50 flex-shrink-0">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
          <Button variant="outline" onClick={handleCopy}>
            Copy to Clipboard
          </Button>
          <Button tone="violet" onClick={onExport} leftIcon={<Download className="h-4 w-4" />}>
            Download YAML
          </Button>
        </div>
      </div>
    </div>
  );
};
