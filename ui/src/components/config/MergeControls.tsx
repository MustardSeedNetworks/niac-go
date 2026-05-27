import {
  AlertCircle,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Download,
  FileCheck,
  RotateCcw,
  Trash2,
} from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { ConfirmModal } from '../../ui/ConfirmModal';
import { Tag } from '../../ui/Tag';
import { H2, SmallText } from '../../ui/Typography';
import { copyToClipboard } from '../../utils/file';
import type { DiffBlock, MergeDecision } from './DiffViewer';
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
  // Confirm modal states
  const [showAcceptLeftConfirm, setShowAcceptLeftConfirm] = useState(false);
  const [showAcceptRightConfirm, setShowAcceptRightConfirm] = useState(false);
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [showClearAllConfirm, setShowClearAllConfirm] = useState(false);

  // Calculate statistics
  const stats = useMemo(() => {
    const changedBlocks = diffBlocks.filter((b) => b.type !== 'unchanged');
    const totalChanges = changedBlocks.length;
    const decisionsCount = mergeDecisions.size;

    let leftCount = 0;
    let rightCount = 0;
    let bothCount = 0;

    for (const decision of mergeDecisions.values()) {
      if (decision.choice === 'left') {
        leftCount++;
      } else if (decision.choice === 'right') {
        rightCount++;
      } else if (decision.choice === 'both') {
        bothCount++;
      }
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
    setShowAcceptLeftConfirm(false);
    onAcceptAllLeft();
  }, [onAcceptAllLeft]);

  const handleAcceptAllRight = useCallback(() => {
    setShowAcceptRightConfirm(false);
    onAcceptAllRight();
  }, [onAcceptAllRight]);

  const handleReset = useCallback(() => {
    setShowResetConfirm(false);
    onReset();
  }, [onReset]);

  const handleClearAll = useCallback(() => {
    setShowClearAllConfirm(false);
    // Reset merge state in place; full page reload would wipe unrelated app state.
    onReset();
  }, [onReset]);

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        {/* Header */}
        <div className="flex-between">
          <H2 className="flex items-center gap-compact">
            <FileCheck className={`${iconSizes.lg} text-brand-accent`} />
            Merge Controls
          </H2>
          {stats.totalChanges > 0 && (
            <div className="flex items-center gap-compact">
              {stats.isComplete ? (
                <Tag colorScheme="green" className="flex items-center gap-tight">
                  <CheckCircle2 className={iconSizes.xs} />
                  All resolved
                </Tag>
              ) : (
                <Tag colorScheme="yellow" className="flex items-center gap-tight">
                  <AlertCircle className={iconSizes.xs} />
                  {stats.totalChanges - stats.decisionsCount} pending
                </Tag>
              )}
            </div>
          )}
        </div>

        {/* Progress bar */}
        {stats.totalChanges > 0 && (
          <div className="stack-sm">
            <div className="flex-between text-sm">
              <SmallText className="text-text-muted">Resolution progress</SmallText>
              <SmallText className="text-text-secondary">
                {stats.decisionsCount} / {stats.totalChanges} ({stats.progress}
                %)
              </SmallText>
            </div>
            <div className="h-2 rounded-full bg-bg-elevated overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-brand-primary to-brand-accent transition-all duration-300"
                style={{ width: `${stats.progress}%` }}
              />
            </div>
          </div>
        )}

        {/* Decision breakdown */}
        {stats.decisionsCount > 0 && (
          <div className="flex items-center gap-comfortable flex-wrap">
            <SmallText className="text-text-muted">Decisions:</SmallText>
            {stats.leftCount > 0 && (
              <Tag colorScheme="blue" className="text-xs">
                <ChevronLeft className={`${iconSizes.xs} mr-1`} />
                {stats.leftCount} left
              </Tag>
            )}
            {stats.rightCount > 0 && (
              <Tag colorScheme="green" className="text-xs">
                <ChevronRight className={`${iconSizes.xs} mr-1`} />
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
        <div className="stack">
          <SmallText className="text-text-muted uppercase tracking-wide font-semibold">
            Quick Actions
          </SmallText>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-default">
            <Button
              variant="outline"
              disabled={disabled || stats.totalChanges === 0}
              onClick={() => setShowAcceptLeftConfirm(true)}
              leftIcon={<ChevronLeft className={iconSizes.md} />}
              className="justify-start"
            >
              Accept all from {leftLabel}
            </Button>

            <Button
              variant="outline"
              disabled={disabled || stats.totalChanges === 0}
              onClick={() => setShowAcceptRightConfirm(true)}
              leftIcon={<ChevronRight className={iconSizes.md} />}
              className="justify-start"
            >
              Accept all from {rightLabel}
            </Button>
          </div>
        </div>

        {/* Main actions */}
        <div className="flex flex-wrap gap-default pt-2 border-t border-surface-border">
          <Button
            tone="violet"
            disabled={disabled || stats.totalChanges === 0}
            onClick={onPreview}
            leftIcon={<FileCheck className={iconSizes.md} />}
          >
            Preview Merged
          </Button>

          <Button
            tone="violet"
            disabled={disabled || !stats.isComplete}
            onClick={onExport}
            leftIcon={<Download className={iconSizes.md} />}
          >
            Export Result
          </Button>

          <Button
            variant="outline"
            disabled={disabled || stats.decisionsCount === 0}
            onClick={() => setShowResetConfirm(true)}
            leftIcon={<RotateCcw className={iconSizes.md} />}
          >
            Reset Decisions
          </Button>

          <Button
            variant="ghost"
            disabled={disabled}
            onClick={() => setShowClearAllConfirm(true)}
            leftIcon={<Trash2 className={iconSizes.md} />}
          >
            Clear All
          </Button>
        </div>

        {/* Help text */}
        <SmallText className="text-text-muted">
          {stats.totalChanges === 0
            ? 'Upload two YAML config files to start comparing and merging.'
            : stats.isComplete
              ? 'All changes resolved. You can now export the merged configuration.'
              : 'Click the merge buttons on each changed block to decide how to merge.'}
        </SmallText>
      </CardContent>

      {/* Accept All Left Confirmation */}
      <ConfirmModal
        isOpen={showAcceptLeftConfirm}
        onConfirm={handleAcceptAllLeft}
        onCancel={() => setShowAcceptLeftConfirm(false)}
        title={`Accept All from ${leftLabel}`}
        message={`Accept all changes from "${leftLabel}"? This will override your current decisions.`}
        confirmLabel="Accept All"
        confirmTone="blue"
      />

      {/* Accept All Right Confirmation */}
      <ConfirmModal
        isOpen={showAcceptRightConfirm}
        onConfirm={handleAcceptAllRight}
        onCancel={() => setShowAcceptRightConfirm(false)}
        title={`Accept All from ${rightLabel}`}
        message={`Accept all changes from "${rightLabel}"? This will override your current decisions.`}
        confirmLabel="Accept All"
        confirmTone="green"
      />

      {/* Reset Confirmation */}
      <ConfirmModal
        isOpen={showResetConfirm}
        onConfirm={handleReset}
        onCancel={() => setShowResetConfirm(false)}
        title="Reset Decisions"
        message="Reset all merge decisions? This cannot be undone."
        confirmLabel="Reset"
        confirmTone="red"
      />

      {/* Clear All Confirmation */}
      <ConfirmModal
        isOpen={showClearAllConfirm}
        onConfirm={handleClearAll}
        onCancel={() => setShowClearAllConfirm(false)}
        title="Clear All"
        message="Clear all files and start over? This will reload the page."
        confirmLabel="Clear All"
        confirmTone="red"
      />
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

export const MergePreviewModal: FC<MergePreviewModalProps> = ({ content, onClose, onExport }) => {
  const lineCount = useMemo(() => content.split('\n').length, [content]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(content);
  }, [content]);

  return (
    <div className="fixed inset-0 z-50 flex-center">
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onClose}
        aria-label="Close modal"
      />
      <div
        className="relative mx-4 max-h-[90vh] w-full max-w-4xl overflow-hidden rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-labelledby="preview-modal-title"
      >
        {/* Modal Header */}
        <div className="flex-between border-b border-surface-border px-6 py-4 flex-shrink-0">
          <div className="flex items-center gap-default">
            <div className="rounded-lg bg-brand-primary/20 pad-xs">
              <FileCheck className={`${iconSizes.lg} text-brand-accent`} />
            </div>
            <div>
              <h2 id="preview-modal-title" className="heading-3 text-text-primary">
                Merged Configuration Preview
              </h2>
              <SmallText className="text-text-muted">{lineCount} lines</SmallText>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg pad-xs text-text-muted hover:bg-surface-hover hover:text-text-primary transition-colors"
            aria-label="Close modal"
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <title>Close</title>
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Modal Content */}
        <div className="flex-1 overflow-auto pad-lg">
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
        <div className="flex justify-end gap-default border-t border-surface-border px-6 py-4 bg-bg-base/50 flex-shrink-0">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
          <Button variant="outline" onClick={handleCopy}>
            Copy to Clipboard
          </Button>
          <Button tone="violet" onClick={onExport} leftIcon={<Download className={iconSizes.md} />}>
            Download YAML
          </Button>
        </div>
      </div>
    </div>
  );
};
