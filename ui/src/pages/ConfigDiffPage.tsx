import { AlertCircle, FileCheck, FileCode, GitCompare, Upload, X } from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import { DiffViewer } from '../components/config/DiffViewer';
import type { DiffBlock, MergeDecision } from '../components/config/diffUtils';
import { computeDiff, generateMergedContent } from '../components/config/diffUtils';
import { MergeControls, MergePreviewModal } from '../components/config/MergeControls';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { H2, P, SmallText } from '../ui/Typography';

interface UploadedFile {
  name: string;
  content: string;
  size: number;
}

/**
 * File upload dropzone component
 */
const FileUploadZone: FC<{
  label: string;
  file: UploadedFile | null;
  onFileUpload: (file: UploadedFile) => void;
  onClear: () => void;
  disabled?: boolean;
}> = ({ label, file, onFileUpload, onClear, disabled }) => {
  const [dragOver, setDragOver] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFile = useCallback(
    async (uploadedFile: File) => {
      setError(null);

      // Validate file type
      if (!uploadedFile.name.match(/\.(yaml|yml)$/i)) {
        setError('Please select a YAML file (.yaml or .yml)');
        return;
      }

      // Validate file size (1MB limit)
      const MaxSize = 1024 * 1024;
      if (uploadedFile.size > MaxSize) {
        setError('File too large. Maximum size is 1MB');
        return;
      }

      try {
        const content = await uploadedFile.text();
        onFileUpload({
          name: uploadedFile.name,
          content,
          size: uploadedFile.size,
        });
      } catch {
        setError('Failed to read file');
      }
    },
    [onFileUpload],
  );

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(false);

      const [droppedFile] = e.dataTransfer.files;
      if (droppedFile) {
        handleFile(droppedFile);
      }
    },
    [handleFile],
  );

  const handleInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const [selectedFile] = e.target.files ?? [];
      if (selectedFile) {
        handleFile(selectedFile);
      }
    },
    [handleFile],
  );

  if (file) {
    return (
      <div className="rounded-xl border border-green-500/30 bg-green-500/10 p-4">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-green-500/20 p-2">
              <FileCode className="h-5 w-5 text-green-300" />
            </div>
            <div>
              <p className="font-semibold text-white">{file.name}</p>
              <SmallText className="text-green-300">
                {formatBytes(file.size)} | {file.content.split('\n').length} lines
              </SmallText>
            </div>
          </div>
          <button
            type="button"
            onClick={onClear}
            disabled={disabled}
            className="rounded-lg p-1.5 text-gray-400 hover:bg-white/10 hover:text-white transition-colors disabled:opacity-50"
            aria-label={`Remove ${file.name}`}
          >
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <label
        className={`block rounded-xl border-2 border-dashed p-6 text-center transition-colors cursor-pointer ${
          dragOver
            ? 'border-violet-400 bg-violet-500/10'
            : 'border-white/20 hover:border-violet-400/50 hover:bg-gray-950/50'
        } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
      >
        <input
          type="file"
          accept=".yaml,.yml"
          onChange={handleInputChange}
          disabled={disabled}
          className="hidden"
        />
        <Upload className="mx-auto h-8 w-8 text-gray-400 mb-2" />
        <p className="text-gray-300 font-medium">{label}</p>
        <SmallText className="text-gray-500">Drag & drop or click to select a YAML file</SmallText>
      </label>
      {error && (
        <div className="flex items-center gap-2 text-red-400 text-sm">
          <AlertCircle className="h-4 w-4" />
          {error}
        </div>
      )}
    </div>
  );
};

/**
 * Format bytes to human-readable string
 */
function formatBytes(size: number): string {
  if (!Number.isFinite(size) || size <= 0) {
    return '0 B';
  }
  const units = ['B', 'KB', 'MB', 'GB'];
  let idx = 0;
  let value = size;
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024;
    idx++;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[idx]}`;
}

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
      <Card className="border-white/5 bg-gray-900/70">
        <CardContent className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div>
              <H2 className="mb-0 flex items-center gap-2">
                <GitCompare className="h-5 w-5 text-violet-300" />
                Config Diff & Merge
              </H2>
              <P className="text-gray-400 mt-1">
                Compare and merge two YAML configuration files with visual diff and merge controls.
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
                  ? 'border border-green-500/30 bg-green-500/10 text-green-300'
                  : 'border border-red-500/30 bg-red-500/10 text-red-300'
              }`}
              role="alert"
            >
              {message.type === 'success' ? (
                <FileCheck className="h-4 w-4" />
              ) : (
                <AlertCircle className="h-4 w-4" />
              )}
              <span>{message.text}</span>
              <button
                type="button"
                onClick={() => setMessage(null)}
                className="ml-auto text-current hover:opacity-70"
                aria-label="Dismiss message"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* File upload section */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="mb-0 text-lg">Original File</H2>
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

        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <H2 className="mb-0 text-lg">Modified File</H2>
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
          <Card className="border-white/5 bg-gray-900/70">
            <CardContent className="space-y-4">
              <H2 className="mb-0 flex items-center gap-2">
                <GitCompare className="h-5 w-5 text-cyan-300" />
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
        </>
      )}

      {/* Empty state when no files */}
      {!hasFiles && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="py-12 text-center">
            <GitCompare className="mx-auto h-12 w-12 text-gray-600" />
            <H2 className="mt-4 mb-2">Ready to Compare</H2>
            <P className="text-gray-400 max-w-md mx-auto">
              Upload two YAML configuration files above to see a side-by-side comparison with
              color-coded changes and interactive merge controls.
            </P>
            <div className="flex flex-wrap justify-center gap-4 mt-6">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded bg-green-500" />
                <SmallText className="text-gray-400">Additions</SmallText>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded bg-red-500" />
                <SmallText className="text-gray-400">Deletions</SmallText>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded bg-yellow-500" />
                <SmallText className="text-gray-400">Modifications</SmallText>
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
