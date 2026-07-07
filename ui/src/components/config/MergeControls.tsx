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
import { useTranslation } from 'react-i18next';
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
export const MergeControls: FC<MergeControlsProps> = ({
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
}) => {
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
                <Tag colorScheme="green" className="flex items-center gap-tight">
                  <CheckCircle2 className={iconSizes.xs} />
                  {t('configDiff.allResolvedTag')}
                </Tag>
              ) : (
                <Tag colorScheme="yellow" className="flex items-center gap-tight">
                  <AlertCircle className={iconSizes.xs} />
                  {t('configDiff.pendingCount', {
                    count: stats.totalChanges - stats.decisionsCount,
                  })}
                </Tag>
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
            className="flex items-start gap-compact rounded-lg border border-status-warning/30 bg-status-warning/10 pad-sm text-sm text-status-warning"
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
              <Tag colorScheme="blue" className="text-xs">
                <ChevronLeft className={`${iconSizes.xs} mr-1`} />
                {t('configDiff.leftCount', { count: stats.leftCount })}
              </Tag>
            )}
            {stats.rightCount > 0 && (
              <Tag colorScheme="green" className="text-xs">
                <ChevronRight className={`${iconSizes.xs} mr-1`} />
                {t('configDiff.rightCount', { count: stats.rightCount })}
              </Tag>
            )}
            {stats.bothCount > 0 && (
              <Tag colorScheme="purple" className="text-xs">
                {t('configDiff.bothCount', { count: stats.bothCount })}
              </Tag>
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
              variant="outline"
              disabled={disabled || stats.totalChanges === 0}
              onClick={() => setShowAcceptLeftConfirm(true)}
              leftIcon={<ChevronLeft className={iconSizes.md} />}
              className="justify-start"
            >
              {t('configDiff.acceptAllFrom', { label: leftLabel })}
            </Button>

            <Button
              variant="outline"
              disabled={disabled || stats.totalChanges === 0}
              onClick={() => setShowAcceptRightConfirm(true)}
              leftIcon={<ChevronRight className={iconSizes.md} />}
              className="justify-start"
            >
              {t('configDiff.acceptAllFrom', { label: rightLabel })}
            </Button>
          </div>
        </div>

        {/* Main actions */}
        <div className="flex flex-wrap gap-default pt-2 border-t border-surface-border">
          <Button
            tone="violet"
            disabled={disabled || stats.totalChanges === 0 || !stats.isComplete}
            onClick={onPreview}
            leftIcon={<FileCheck className={iconSizes.md} />}
          >
            {t('configDiff.previewMergedButton')}
          </Button>

          <Button
            tone="violet"
            disabled={disabled || !stats.isComplete}
            onClick={onExport}
            leftIcon={<Download className={iconSizes.md} />}
          >
            {t('configDiff.exportResultButton')}
          </Button>

          <Button
            variant="outline"
            disabled={disabled || stats.decisionsCount === 0}
            onClick={() => setShowResetConfirm(true)}
            leftIcon={<RotateCcw className={iconSizes.md} />}
          >
            {t('configDiff.resetDecisionsButton')}
          </Button>

          <Button
            variant="ghost"
            disabled={disabled}
            onClick={() => setShowClearAllConfirm(true)}
            leftIcon={<Trash2 className={iconSizes.md} />}
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
      <ConfirmModal
        isOpen={showAcceptLeftConfirm}
        onConfirm={handleAcceptAllLeft}
        onCancel={() => setShowAcceptLeftConfirm(false)}
        title={t('configDiff.acceptAllFromTitle', { label: leftLabel })}
        message={t('configDiff.acceptAllFromMessage', { label: leftLabel })}
        confirmLabel={t('configDiff.acceptAllConfirmLabel')}
        confirmTone="blue"
      />

      {/* Accept All Right Confirmation */}
      <ConfirmModal
        isOpen={showAcceptRightConfirm}
        onConfirm={handleAcceptAllRight}
        onCancel={() => setShowAcceptRightConfirm(false)}
        title={t('configDiff.acceptAllFromTitle', { label: rightLabel })}
        message={t('configDiff.acceptAllFromMessage', { label: rightLabel })}
        confirmLabel={t('configDiff.acceptAllConfirmLabel')}
        confirmTone="green"
      />

      {/* Reset Confirmation */}
      <ConfirmModal
        isOpen={showResetConfirm}
        onConfirm={handleReset}
        onCancel={() => setShowResetConfirm(false)}
        title={t('configDiff.resetDecisionsButton')}
        message={t('configDiff.resetConfirmMessage')}
        confirmLabel={t('configDiff.resetConfirmLabel')}
        confirmTone="red"
      />

      {/* Clear All Confirmation */}
      <ConfirmModal
        isOpen={showClearAllConfirm}
        onConfirm={handleClearAll}
        onCancel={() => setShowClearAllConfirm(false)}
        title={t('configDiff.clearAllButton')}
        message={t('configDiff.clearAllConfirmMessage')}
        confirmLabel={t('configDiff.clearAllButton')}
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
  const { t } = useTranslation('pages');
  const lineCount = useMemo(() => content.split('\n').length, [content]);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(content);
  }, [content]);

  return (
    <div className="fixed inset-0 z-50 flex-center">
      <button
        type="button"
        className="absolute inset-0 bg-scrim/70 backdrop-blur-sm"
        onClick={onClose}
        aria-label={t('configDiff.closeModalLabel')}
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
                {t('configDiff.previewModalTitle')}
              </h2>
              <SmallText className="text-text-muted">
                {t('configDiff.lineCount', { count: lineCount })}
              </SmallText>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg pad-xs text-text-muted hover:bg-surface-hover hover:text-text-primary transition-colors"
            aria-label={t('configDiff.closeModalLabel')}
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <title>{t('configDiff.closeButton')}</title>
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
            {t('configDiff.closeButton')}
          </Button>
          <Button variant="outline" onClick={handleCopy}>
            {t('configDiff.copyToClipboardButton')}
          </Button>
          <Button tone="violet" onClick={onExport} leftIcon={<Download className={iconSizes.md} />}>
            {t('configDiff.downloadYamlButton')}
          </Button>
        </div>
      </div>
    </div>
  );
};
