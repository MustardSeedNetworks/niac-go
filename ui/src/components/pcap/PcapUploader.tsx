import { AlertCircle, CheckCircle, FileUp, Upload, X } from 'lucide-react';
import { type FC, useCallback, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import type { TFunction } from '../../i18n';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { formatBytes } from '../../utils/format';

interface PcapUploaderProps {
  onFileSelect: (file: File | null) => void;
  onAnalyze: () => void;
  onValidationError: (message: string) => void;
  isAnalyzing: boolean;
  selectedFile: File | null;
  error: string | null;
  success: string | null;
  /**
   * Upload completion percent (0-100) while the file is in flight, or
   * `null` when no upload is in progress. The upload itself happens in
   * the parent page (via the XHR-based uploadPcapWithProgress client
   * call, which is what makes onprogress observable); this component
   * only renders the resulting determinate progress bar.
   */
  uploadProgress: number | null;
}

/** Maximum file size: 100MB */
const MAX_FILE_SIZE = 100 * 1024 * 1024;

/** Accepted file extensions */
const ACCEPTED_EXTENSIONS = ['.pcap', '.pcapng', '.cap'];

/**
 * Validate file for PCAP upload
 */
function validateFile(file: File, t: TFunction<'pages'>): { valid: boolean; error?: string } {
  // Check file extension
  const fileName = file.name.toLowerCase();
  const hasValidExtension = ACCEPTED_EXTENSIONS.some((ext) => fileName.endsWith(ext));

  if (!hasValidExtension) {
    return {
      valid: false,
      error: t('libraryPcaps.uploader.invalidFileType', {
        extensions: ACCEPTED_EXTENSIONS.join(', '),
      }),
    };
  }

  // Check file size
  if (file.size > MAX_FILE_SIZE) {
    return {
      valid: false,
      error: t('libraryPcaps.uploader.fileTooLarge', { size: formatBytes(MAX_FILE_SIZE) }),
    };
  }

  if (file.size === 0) {
    return {
      valid: false,
      error: t('libraryPcaps.uploader.fileEmpty'),
    };
  }

  return { valid: true };
}

/**
 * PCAP File Uploader Component
 *
 * Provides drag-and-drop and click-to-select file upload for PCAP files.
 * Validates file type and size before allowing analysis.
 */
export const PcapUploader: FC<PcapUploaderProps> = ({
  onFileSelect,
  onAnalyze,
  onValidationError,
  isAnalyzing,
  selectedFile,
  error,
  success,
  uploadProgress,
}) => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Handle file selection from input
  const handleFileChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) {
        return;
      }

      const validation = validateFile(file, t);
      if (!validation.valid) {
        // Reset input
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        onValidationError(validation.error ?? t('libraryPcaps.uploader.invalidFile'));
        return;
      }

      onFileSelect(file);
    },
    [onFileSelect, onValidationError, t],
  );

  // Handle drag events
  const handleDragOver = useCallback((event: React.DragEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((event: React.DragEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragOver(false);
  }, []);

  const handleDrop = useCallback(
    (event: React.DragEvent<HTMLButtonElement>) => {
      event.preventDefault();
      event.stopPropagation();
      setIsDragOver(false);

      const file = event.dataTransfer.files?.[0];
      if (!file) {
        return;
      }

      const validation = validateFile(file, t);
      if (!validation.valid) {
        onValidationError(validation.error ?? t('libraryPcaps.uploader.invalidFile'));
        return;
      }

      onFileSelect(file);
    },
    [onFileSelect, onValidationError, t],
  );

  // Handle click to open file dialog
  const handleClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  // Clear selected file
  const handleClear = useCallback(() => {
    onFileSelect(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [onFileSelect]);

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack-lg">
        {/* Drag and Drop Zone */}
        <input
          ref={fileInputRef}
          type="file"
          accept=".pcap,.pcapng,.cap"
          onChange={handleFileChange}
          className="hidden"
        />
        <button
          type="button"
          onClick={handleClick}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          className={`
            relative cursor-pointer rounded-xl border-2 border-dashed pad-xl text-center transition-all
            ${
              isDragOver
                ? 'border-brand-accent bg-brand-primary/20'
                : 'border-surface-border bg-bg-base/40 hover:border-surface-border hover:bg-bg-base/60'
            }
            ${isAnalyzing ? 'pointer-events-none opacity-50' : ''}
          `}
          aria-label={t('libraryPcaps.uploader.dropAriaLabel')}
        >
          <div className="flex flex-col items-center gap-default">
            <div
              className={`rounded-full pad ${isDragOver ? 'bg-brand-primary/20' : 'bg-bg-elevated/50'}`}
            >
              <Upload
                className={`${iconSizes['2xl']} ${isDragOver ? 'text-brand-accent' : 'text-text-muted'}`}
              />
            </div>

            <div>
              <p className="text-lg font-medium text-text-primary">
                {isDragOver
                  ? t('libraryPcaps.uploader.dropToUpload')
                  : t('libraryPcaps.uploader.dragDropTitle')}
              </p>
              <SmallText className="text-text-muted">
                {t('libraryPcaps.uploader.browseHint', {
                  extensions: ACCEPTED_EXTENSIONS.join(', '),
                })}
              </SmallText>
            </div>

            <SmallText className="text-text-muted">
              {t('libraryPcaps.uploader.maxFileSizeLabel', { size: formatBytes(MAX_FILE_SIZE) })}
            </SmallText>
          </div>
        </button>

        {/* Selected File Display */}
        {selectedFile && (
          <div className="flex-between rounded-lg border border-surface-border bg-bg-base/50 pad">
            <div className="flex items-center gap-default">
              <FileUp className={`${iconSizes.lg} text-brand-accent`} />
              <div>
                <p className="font-medium text-text-primary">{selectedFile.name}</p>
                <SmallText className="text-text-muted">{formatBytes(selectedFile.size)}</SmallText>
              </div>
              <Tag colorScheme="purple" className="text-xs">
                {tCommon('status.ready')}
              </Tag>
            </div>

            <div className="flex items-center gap-compact">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClear}
                disabled={isAnalyzing}
                leftIcon={<X className={iconSizes.md} />}
              >
                {tCommon('buttons.clear')}
              </Button>
            </div>
          </div>
        )}

        {/* Upload Progress */}
        {uploadProgress !== null && (
          <div className="stack-sm" data-testid="pcap-upload-progress">
            <div className="flex-between">
              <SmallText className="text-text-muted">
                {t('libraryPcaps.uploader.uploadingProgress', { percent: uploadProgress })}
              </SmallText>
            </div>
            <div
              className="h-2 w-full overflow-hidden rounded-full bg-bg-base/60"
              role="progressbar"
              aria-valuenow={uploadProgress}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div
                className="h-full rounded-full bg-brand-accent transition-[width] duration-150"
                style={{ width: `${uploadProgress}%` }}
                data-testid="pcap-upload-progress-bar"
              />
            </div>
          </div>
        )}

        {/* Status Messages */}
        {error && (
          <div className="flex items-center gap-compact rounded-lg border border-status-error/30 bg-status-error/20 pad-sm">
            <AlertCircle className={`${iconSizes.lg} flex-shrink-0 text-status-error`} />
            <SmallText className="text-status-error">{error}</SmallText>
          </div>
        )}

        {success && (
          <div className="flex items-center gap-compact rounded-lg border border-status-success/30 bg-status-success/20 pad-sm">
            <CheckCircle className={`${iconSizes.lg} flex-shrink-0 text-status-success`} />
            <SmallText className="text-status-success">{success}</SmallText>
          </div>
        )}

        {/* Analyze Button */}
        <Button
          tone="violet"
          size="lg"
          className="w-full"
          onClick={onAnalyze}
          disabled={!selectedFile || isAnalyzing}
          leftIcon={
            isAnalyzing ? (
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-surface-border border-t-white" />
            ) : (
              <FileUp className={iconSizes.lg} />
            )
          }
        >
          {isAnalyzing
            ? t('libraryPcaps.uploader.analyzing')
            : t('libraryPcaps.uploader.analyzeButton')}
        </Button>
      </CardContent>
    </Card>
  );
};

export default PcapUploader;
