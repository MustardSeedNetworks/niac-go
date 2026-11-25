import {
  AlertCircle,
  ChevronLeft,
  ChevronRight,
  Download,
  FileCheck,
  RotateCcw,
  Trash2,
} from 'lucide-react';
import { useCallback, useId, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../shared/Button';
import { CategoryChip } from '../../shared/CategoryChip';
import { ConfirmDialog } from '../../shared/ConfirmDialog';
import { StatusPill } from '../../shared/StatusPill';
import { Card, CardContent } from '../../ui/Card';
import { useDialogLabels } from '../../ui/dialogLabels';
import { Modal } from '../../ui/Modal';
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
  onClearAll: () => void;
  onExport: () => void;
  onPreview: () => void;
  disabled?: boolean;
  leftLabel?: string;
  rightLabel?: string;
}

/**
 * Merge action controls for the config diff page
 */
export function MergeControls({
  diffBlocks,
  mergeDecisions,
  onAcceptAllLeft,
  onAcceptAllRight,
  onReset,
  onClearAll,
  onExport,
  onPreview,
  disabled = false,
  leftLabel = 'Original',
  rightLabel = 'Modified',
}: MergeControlsProps) {
  const dialogLabels = useDialogLabels();
  const { t } = useTranslation('pages');
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
    onClearAll();
  }, [onClearAll]);

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        {/* Header */}
        <div className="flex-between">
          <H2 className="flex items-center gap-compact">
            <FileCheck className={`${iconSizes.lg} text-brand-accent`} />
            {t('configDiff.mergeControlsTitle')}
          </H2>
          {stats.totalChanges > 0 && (
            <div className="flex items-center gap-compact">
              {stats.isComplete ? (
                <StatusPill tone="success" label={t('configDiff.allResolvedTag')} />
              ) : (
                <StatusPill
                  tone="warning"
                  label={t('configDiff.pendingCount', {
                    count: stats.totalChanges - stats.decisionsCount,
                  })}
                />
              )}
            </div>
          )}
        </div>

        {/* Progress bar */}
        {stats.totalChanges > 0 && (
          <div className="stack-sm">
            <div className="flex-between text-sm">
              <SmallText className="text-text-muted">
                {t('configDiff.resolutionProgress')}
              </SmallText>
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

        {/* Undecided-blocks warning — a "modified" block with no Left/Both/
            Right decision silently keeps the original (left) content in
            the preview, export, and server-side merge. Surface that risk
            explicitly instead of letting it happen quietly. */}
        {stats.totalChanges > 0 && !stats.isComplete && (
          <div
            className="flex items-start gap-compact rounded-lg border border-status-warning/30 bg-status-warning/10 pad-sm text-sm text-status-warning-strong"
            role="alert"
          >
            <AlertCircle className={`${iconSizes.md} mt-0.5 flex-shrink-0`} />
            <span>
              {t('configDiff.undecidedBlocksWarning', {
                count: stats.totalChanges - stats.decisionsCount,
              })}
            </span>
          </div>
        )}

        {/* Decision breakdown */}
        {stats.decisionsCount > 0 && (
          <div className="flex items-center gap-comfortable flex-wrap">
            <SmallText className="text-text-muted">{t('configDiff.decisionsLabel')}</SmallText>
            {stats.leftCount > 0 && (
              <CategoryChip
                icon={<ChevronLeft className="h-3.5 w-3.5" />}
                label={t('configDiff.leftCount', { count: stats.leftCount })}
              />
            )}
            {stats.rightCount > 0 && (
              <CategoryChip
                icon={<ChevronRight className="h-3.5 w-3.5" />}
                label={t('configDiff.rightCount', { count: stats.rightCount })}
              />
            )}
            {stats.bothCount > 0 && (
              <CategoryChip label={t('configDiff.bothCount', { count: stats.bothCount })} />
            )}
          </div>
        )}

        {/* Quick actions */}
        <div className="stack">
          <SmallText className="text-text-muted uppercase tracking-wide font-semibold">
            {t('configDiff.quickActionsLabel')}
          </SmallText>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-default">
            <Button
              variant="secondary"
              disabled={disabled || stats.totalChanges === 0}
              onClick={() => setShowAcceptLeftConfirm(true)}
              icon={<ChevronLeft className={iconSizes.md} />}
              className="justify-start"
            >
              {t('configDiff.acceptAllFrom', { label: leftLabel })}
            </Button>

            <Button
              variant="secondary"
              disabled={disabled || stats.totalChanges === 0}
              onClick={() => setShowAcceptRightConfirm(true)}
              icon={<ChevronRight className={iconSizes.md} />}
              className="justify-start"
            >
              {t('configDiff.acceptAllFrom', { label: rightLabel })}
            </Button>
          </div>
        </div>

        {/* Main actions */}
        <div className="flex flex-wrap gap-default pt-2 border-t border-surface-border">
          <Button
            variant="primary"
            disabled={disabled || stats.totalChanges === 0 || !stats.isComplete}
            onClick={onPreview}
            icon={<FileCheck className={iconSizes.md} />}
          >
            {t('configDiff.previewMergedButton')}
          </Button>

          <Button
            variant="primary"
            disabled={disabled || !stats.isComplete}
            onClick={onExport}
            icon={<Download className={iconSizes.md} />}
          >
            {t('configDiff.exportResultButton')}
          </Button>

          <Button
            variant="secondary"
            disabled={disabled || stats.decisionsCount === 0}
            onClick={() => setShowResetConfirm(true)}
            icon={<RotateCcw className={iconSizes.md} />}
          >
            {t('configDiff.resetDecisionsButton')}
          </Button>

          <Button
            variant="ghost"
            disabled={disabled}
            onClick={() => setShowClearAllConfirm(true)}
            icon={<Trash2 className={iconSizes.md} />}
          >
            {t('configDiff.clearAllButton')}
          </Button>
        </div>

        {/* Help text */}
        <SmallText className="text-text-muted">
          {stats.totalChanges === 0
            ? t('configDiff.helpTextEmpty')
            : stats.isComplete
              ? t('configDiff.helpTextComplete')
              : t('configDiff.resolveAllBeforePreview')}
        </SmallText>
      </CardContent>

      {/* Accept All Left Confirmation */}
      <ConfirmDialog
        {...dialogLabels}
        open={showAcceptLeftConfirm}
        onConfirm={handleAcceptAllLeft}
        onCancel={() => setShowAcceptLeftConfirm(false)}
        title={t('configDiff.acceptAllFromTitle', { label: leftLabel })}
        message={t('configDiff.acceptAllFromMessage', { label: leftLabel })}
        confirmLabel={t('configDiff.acceptAllConfirmLabel')}
        variant="primary"
      />

      {/* Accept All Right Confirmation */}
      <ConfirmDialog
        {...dialogLabels}
        open={showAcceptRightConfirm}
        onConfirm={handleAcceptAllRight}
        onCancel={() => setShowAcceptRightConfirm(false)}
        title={t('configDiff.acceptAllFromTitle', { label: rightLabel })}
        message={t('configDiff.acceptAllFromMessage', { label: rightLabel })}
        confirmLabel={t('configDiff.acceptAllConfirmLabel')}
        variant="primary"
      />

      {/* Reset Confirmation */}
      <ConfirmDialog
        {...dialogLabels}
        open={showResetConfirm}
        onConfirm={handleReset}
        onCancel={() => setShowResetConfirm(false)}
        title={t('configDiff.resetDecisionsButton')}
        message={t('configDiff.resetConfirmMessage')}
        confirmLabel={t('configDiff.resetConfirmLabel')}
      />

      {/* Clear All Confirmation */}
      <ConfirmDialog
        {...dialogLabels}
        open={showClearAllConfirm}
        onConfirm={handleClearAll}
        onCancel={() => setShowClearAllConfirm(false)}
        title={t('configDiff.clearAllButton')}
        message={t('configDiff.clearAllConfirmMessage')}
        confirmLabel={t('configDiff.clearAllButton')}
      />
    </Card>
  );
}

/**
 * Preview modal for merged content
 */
interface MergePreviewModalProps {
  content: string;
  onClose: () => void;
  onExport: () => void;
}

export function MergePreviewModal({ content, onClose, onExport }: MergePreviewModalProps) {
  const { t } = useTranslation('pages');
  const titleId = useId();
  const lineCount = useMemo(() => content.split('\n').length, [content]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(content);
  }, [content]);

  return (
    <Modal
      isOpen
      onClose={onClose}
      size="full"
      labelledBy={titleId}
      header={
        <div className="flex items-center gap-default">
          <div className="rounded-lg bg-brand-primary/20 pad-xs">
            <FileCheck className={`${iconSizes.lg} text-brand-accent`} />
          </div>
          <div>
            <h2 id={titleId} className="heading-3 text-text-primary">
              {t('configDiff.previewModalTitle')}
            </h2>
            <SmallText className="text-text-muted">
              {t('configDiff.lineCount', { count: lineCount })}
            </SmallText>
          </div>
        </div>
      }
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            {t('configDiff.closeButton')}
          </Button>
          <Button variant="secondary" onClick={handleCopy}>
            {t('configDiff.copyToClipboardButton')}
          </Button>
          <Button variant="primary" onClick={onExport} icon={<Download className={iconSizes.md} />}>
            {t('configDiff.downloadYamlButton')}
          </Button>
        </>
      }
    >
      <YamlViewer
        ariaLabel={t('configDiff.mergePreviewEditorAria')}
        value={content}
        height="100%"
        minHeight="300px"
        maxHeight="60vh"
        showLineNumbers={true}
        showFoldGutter={true}
      />
    </Modal>
  );
}
