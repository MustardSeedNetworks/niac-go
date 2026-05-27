import { AlertCircle, FileCheck, GitCompare, Layers, X } from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { mergeConfigs as apiMergeConfigs } from '../api/client';
import {
  computeDiff,
  type DiffBlock,
  DiffViewer,
  generateMergedContent,
  type MergeDecision,
} from '../components/config/DiffViewer';
import { MergeControls, MergePreviewModal } from '../components/config/MergeControls';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, P, SmallText } from '../ui/Typography';
import { FileUploadZone, type UploadedFile } from './config-diff/FileUploadZone';

/**
 * Config Diff/Merge Page
 *
 * Allows users to:
 * - Upload two YAML config files
 * - View side-by-side diff with syntax highlighting
 * - Make merge decisions for each change
 * - Export the merged result
 */
export const ConfigDiffPage: FC = () => {
  const { t } = useTranslation('pages');
  // File state
  const [leftFile, setLeftFile] = useState<UploadedFile | null>(null);
  const [rightFile, setRightFile] = useState<UploadedFile | null>(null);

  // Merge state
  const [mergeDecisions, setMergeDecisions] = useState<Map<string, MergeDecision>>(new Map());

  // UI state
  const [message, setMessage] = useState<{
    type: 'success' | 'error';
    text: string;
  } | null>(null);
  const [showPreview, setShowPreview] = useState(false);

  // Compute diff blocks for merge controls
  const diffBlocks = useMemo<DiffBlock[]>(() => {
    if (!(leftFile && rightFile)) {
      return [];
    }
    return computeDiff(leftFile.content, rightFile.content);
  }, [leftFile, rightFile]);

  // Generate merged content
  const mergedContent = useMemo(() => {
    if (!(leftFile && rightFile)) {
      return '';
    }
    return generateMergedContent(leftFile.content, rightFile.content, mergeDecisions);
  }, [leftFile, rightFile, mergeDecisions]);

  // Handle file uploads
  const handleLeftFileUpload = useCallback((file: UploadedFile) => {
    setLeftFile(file);
    setMergeDecisions(new Map());
    setMessage(null);
  }, []);

  const handleRightFileUpload = useCallback((file: UploadedFile) => {
    setRightFile(file);
    setMergeDecisions(new Map());
    setMessage(null);
  }, []);

  // Handle merge decisions
  const handleMergeDecision = useCallback((blockId: string, choice: MergeDecision['choice']) => {
    setMergeDecisions((prev) => {
      const next = new Map(prev);
      next.set(blockId, { blockId, choice });
      return next;
    });
  }, []);

  // Accept all from left
  const handleAcceptAllLeft = useCallback(() => {
    const changedBlocks = diffBlocks.filter((b) => b.type !== 'unchanged');
    const decisions = new Map<string, MergeDecision>();
    for (const block of changedBlocks) {
      decisions.set(block.id, { blockId: block.id, choice: 'left' });
    }
    setMergeDecisions(decisions);
    setMessage({
      type: 'success',
      text: 'All changes set to accept from left file',
    });
  }, [diffBlocks]);

  // Accept all from right
  const handleAcceptAllRight = useCallback(() => {
    const changedBlocks = diffBlocks.filter((b) => b.type !== 'unchanged');
    const decisions = new Map<string, MergeDecision>();
    for (const block of changedBlocks) {
      decisions.set(block.id, { blockId: block.id, choice: 'right' });
    }
    setMergeDecisions(decisions);
    setMessage({
      type: 'success',
      text: 'All changes set to accept from right file',
    });
  }, [diffBlocks]);

  // Reset all decisions
  const handleReset = useCallback(() => {
    setMergeDecisions(new Map());
    setMessage({ type: 'success', text: 'All merge decisions reset' });
  }, []);

  // Export merged content
  const handleExport = useCallback(() => {
    if (!mergedContent) {
      return;
    }

    const blob = new Blob([mergedContent], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;

    // Generate filename from source files
    const baseName = leftFile?.name.replace(/\.(yaml|yml)$/i, '') || 'config';
    link.download = `${baseName}-merged.yaml`;

    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);

    setMessage({
      type: 'success',
      text: 'Merged configuration exported successfully',
    });
  }, [mergedContent, leftFile]);

  // Show preview modal
  const handlePreview = useCallback(() => {
    setShowPreview(true);
  }, []);

  // Check if both files are loaded
  const hasFiles = leftFile !== null && rightFile !== null;

  return (
    <div className="space-y-6">
      {/* Header section */}
      <Card className="border-surface-border bg-bg-surface/70">
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <H2 className="flex items-center gap-2">
                <GitCompare className={`${iconSizes.lg} text-brand-accent`} />
                Compare & Merge
              </H2>
              <P className="text-text-muted mt-1">
                Compare two YAML network configs side-by-side and merge changes between them.
              </P>
            </div>
            {hasFiles && (
              <div className="flex items-center gap-2">
                <Tag colorScheme="purple">
                  {diffBlocks.filter((b) => b.type !== 'unchanged').length} changes
                </Tag>
              </div>
            )}
          </div>

          {/* Status message */}
          {message && (
            <div
              className={`flex items-center gap-2 rounded-lg p-3 ${
                message.type === 'success'
                  ? 'border border-status-success/30 bg-status-success/10 text-status-success'
                  : 'border border-status-error/30 bg-status-error/10 text-status-error'
              }`}
              role="alert"
            >
              {message.type === 'success' ? (
                <FileCheck className={iconSizes.md} />
              ) : (
                <AlertCircle className={iconSizes.md} />
              )}
              <span>{message.text}</span>
              <button
                type="button"
                onClick={() => setMessage(null)}
                className="ml-auto text-current hover:opacity-70"
                aria-label="Dismiss message"
              >
                <X className={iconSizes.md} />
              </button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* File upload section */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="text-lg">{t('configDiff.originalFileTitle')}</H2>
              {leftFile && (
                <Tag colorScheme="blue" className="text-xs">
                  Source
                </Tag>
              )}
            </div>
            <FileUploadZone
              label="Upload original config"
              file={leftFile}
              onFileUpload={handleLeftFileUpload}
              onClear={() => {
                setLeftFile(null);
                setMergeDecisions(new Map());
              }}
            />
          </CardContent>
        </Card>

        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="text-lg">{t('configDiff.modifiedFileTitle')}</H2>
              {rightFile && (
                <Tag colorScheme="green" className="text-xs">
                  Changes
                </Tag>
              )}
            </div>
            <FileUploadZone
              label="Upload modified config"
              file={rightFile}
              onFileUpload={handleRightFileUpload}
              onClear={() => {
                setRightFile(null);
                setMergeDecisions(new Map());
              }}
            />
          </CardContent>
        </Card>
      </div>

      {/* Diff viewer section */}
      {hasFiles && (
        <>
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent className="space-y-4">
              <H2 className="flex items-center gap-2">
                <GitCompare className={`${iconSizes.lg} text-status-info`} />
                Side-by-Side Comparison
              </H2>
              <DiffViewer
                leftContent={leftFile.content}
                rightContent={rightFile.content}
                leftLabel={leftFile.name}
                rightLabel={rightFile.name}
                mergeDecisions={mergeDecisions}
                onMergeDecision={handleMergeDecision}
                showMergeControls={true}
              />
            </CardContent>
          </Card>

          {/* Merge controls */}
          <MergeControls
            diffBlocks={diffBlocks}
            mergeDecisions={mergeDecisions}
            onAcceptAllLeft={handleAcceptAllLeft}
            onAcceptAllRight={handleAcceptAllRight}
            onReset={handleReset}
            onExport={handleExport}
            onPreview={handlePreview}
            leftLabel={leftFile.name}
            rightLabel={rightFile.name}
          />

          {/* Server-side overlay merge (CLI parity) */}
          <Card className="border-surface-border bg-bg-surface/70">
            <CardContent className="space-y-3">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <H2 className="mb-1 flex items-center gap-2 text-lg">
                    <Layers className={`${iconSizes.lg} text-brand-accent`} />
                    Server-side overlay merge
                  </H2>
                  <P className="text-sm text-text-muted">
                    Treats the right file as an overlay applied to the left base — same semantics as{' '}
                    <code>niac config merge</code>. Devices in the overlay with the same name
                    REPLACE base entries; overlay-only devices are appended; base-only devices are
                    kept. Useful when you want CLI-equivalent output without resolving each diff
                    block manually.
                  </P>
                </div>
                <Button
                  tone="violet"
                  onClick={async () => {
                    if (!(leftFile && rightFile)) return;
                    try {
                      const result = await apiMergeConfigs({
                        base: leftFile.content,
                        overlay: rightFile.content,
                      });
                      const blob = new Blob([result.merged], { type: 'application/x-yaml' });
                      const url = URL.createObjectURL(blob);
                      const link = document.createElement('a');
                      link.href = url;
                      const baseName = leftFile.name.replace(/\.(yaml|yml)$/i, '');
                      link.download = `${baseName}-overlay-merged.yaml`;
                      link.click();
                      URL.revokeObjectURL(url);
                      setMessage({
                        type: 'success',
                        text: `Server-side merge: ${result.baseDevices} base + ${result.overlayDevices} overlay → ${result.mergedDevices} devices`,
                      });
                    } catch (err) {
                      setMessage({
                        type: 'error',
                        text: `Server-side merge failed: ${(err as Error).message}`,
                      });
                    }
                  }}
                >
                  Run server merge
                </Button>
              </div>
            </CardContent>
          </Card>
        </>
      )}

      {/* Empty state when no files */}
      {!hasFiles && (
        <Card className="border-surface-border bg-bg-surface/70">
          <CardContent className="py-12 text-center">
            <GitCompare className={`mx-auto ${iconSizes['3xl']} text-text-disabled`} />
            <H2 className="mt-4 mb-2">{t('configDiff.readyToCompareTitle')}</H2>
            <P className="text-text-muted max-w-md mx-auto">
              Upload two YAML configuration files above to see a side-by-side comparison with
              color-coded changes and interactive merge controls.
            </P>
            <div className="flex flex-wrap justify-center gap-4 mt-6">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded bg-status-success" />
                <SmallText className="text-text-muted">Additions</SmallText>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded bg-status-error" />
                <SmallText className="text-text-muted">Deletions</SmallText>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded bg-status-warning" />
                <SmallText className="text-text-muted">Modifications</SmallText>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Preview modal */}
      {showPreview && (
        <MergePreviewModal
          content={mergedContent}
          onClose={() => setShowPreview(false)}
          onExport={handleExport}
        />
      )}
    </div>
  );
};
