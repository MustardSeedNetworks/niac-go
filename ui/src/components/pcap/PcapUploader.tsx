import { type FC, useState, useCallback, useRef } from 'react';
import { Card, CardContent, Button, SmallText, Tag } from '../../ui';
import { Upload, FileUp, X, AlertCircle, CheckCircle } from 'lucide-react';

interface PcapUploaderProps {
  onFileSelect: (file: File) => void;
  onAnalyze: () => void;
  isAnalyzing: boolean;
  selectedFile: File | null;
  error: string | null;
  success: string | null;
}

/** Maximum file size: 100MB */
const MAX_FILE_SIZE = 100 * 1024 * 1024;

/** Accepted file extensions */
const ACCEPTED_EXTENSIONS = ['.pcap', '.pcapng', '.cap'];

/**
 * Format bytes to human-readable string
 */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

/**
 * Validate file for PCAP upload
 */
function validateFile(file: File): { valid: boolean; error?: string } {
  // Check file extension
  const fileName = file.name.toLowerCase();
  const hasValidExtension = ACCEPTED_EXTENSIONS.some((ext) => fileName.endsWith(ext));

  if (!hasValidExtension) {
    return {
      valid: false,
      error: `Invalid file type. Please select a PCAP file (${ACCEPTED_EXTENSIONS.join(', ')})`,
    };
  }

  // Check file size
  if (file.size > MAX_FILE_SIZE) {
    return {
      valid: false,
      error: `File too large. Maximum size is ${formatBytes(MAX_FILE_SIZE)}`,
    };
  }

  if (file.size === 0) {
    return {
      valid: false,
      error: 'File is empty',
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
  isAnalyzing,
  selectedFile,
  error,
  success,
}) => {
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Handle file selection from input
  const handleFileChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;

      const validation = validateFile(file);
      if (!validation.valid) {
        // Reset input
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        return;
      }

      onFileSelect(file);
    },
    [onFileSelect]
  );

  // Handle drag events
  const handleDragOver = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((event: React.DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsDragOver(false);
  }, []);

  const handleDrop = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      event.stopPropagation();
      setIsDragOver(false);

      const file = event.dataTransfer.files?.[0];
      if (!file) return;

      const validation = validateFile(file);
      if (!validation.valid) {
        return;
      }

      onFileSelect(file);
    },
    [onFileSelect]
  );

  // Handle click to open file dialog
  const handleClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  // Clear selected file
  const handleClear = useCallback(() => {
    onFileSelect(null as unknown as File);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, [onFileSelect]);

  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        {/* Drag and Drop Zone */}
        <div
          role="button"
          tabIndex={0}
          onClick={handleClick}
          onKeyDown={(e) => e.key === 'Enter' && handleClick()}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          className={`
            relative cursor-pointer rounded-xl border-2 border-dashed p-8 text-center transition-all
            ${
              isDragOver
                ? 'border-violet-400 bg-violet-900/20'
                : 'border-white/10 bg-gray-950/40 hover:border-white/20 hover:bg-gray-950/60'
            }
            ${isAnalyzing ? 'pointer-events-none opacity-50' : ''}
          `}
          aria-label="Drop PCAP file here or click to select"
        >
          <input
            ref={fileInputRef}
            type="file"
            accept=".pcap,.pcapng,.cap"
            onChange={handleFileChange}
            className="hidden"
            aria-hidden="true"
          />

          <div className="flex flex-col items-center gap-3">
            <div
              className={`rounded-full p-4 ${
                isDragOver ? 'bg-violet-500/20' : 'bg-gray-800/50'
              }`}
            >
              <Upload
                className={`h-8 w-8 ${
                  isDragOver ? 'text-violet-400' : 'text-gray-400'
                }`}
              />
            </div>

            <div>
              <p className="text-lg font-medium text-white">
                {isDragOver ? 'Drop file to upload' : 'Drag & drop PCAP file'}
              </p>
              <SmallText className="text-gray-400">
                or click to browse ({ACCEPTED_EXTENSIONS.join(', ')})
              </SmallText>
            </div>

            <SmallText className="text-gray-500">
              Maximum file size: {formatBytes(MAX_FILE_SIZE)}
            </SmallText>
          </div>
        </div>

        {/* Selected File Display */}
        {selectedFile && (
          <div className="flex items-center justify-between rounded-lg border border-white/10 bg-gray-950/50 p-4">
            <div className="flex items-center gap-3">
              <FileUp className="h-5 w-5 text-violet-400" />
              <div>
                <p className="font-medium text-white">{selectedFile.name}</p>
                <SmallText className="text-gray-400">
                  {formatBytes(selectedFile.size)}
                </SmallText>
              </div>
              <Tag colorScheme="purple" className="text-xs">
                Ready
              </Tag>
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClear}
                disabled={isAnalyzing}
                leftIcon={<X className="h-4 w-4" />}
              >
                Clear
              </Button>
            </div>
          </div>
        )}

        {/* Status Messages */}
        {error && (
          <div className="flex items-center gap-2 rounded-lg border border-red-500/30 bg-red-900/20 p-3">
            <AlertCircle className="h-5 w-5 flex-shrink-0 text-red-400" />
            <SmallText className="text-red-300">{error}</SmallText>
          </div>
        )}

        {success && (
          <div className="flex items-center gap-2 rounded-lg border border-green-500/30 bg-green-900/20 p-3">
            <CheckCircle className="h-5 w-5 flex-shrink-0 text-green-400" />
            <SmallText className="text-green-300">{success}</SmallText>
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
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
            ) : (
              <FileUp className="h-5 w-5" />
            )
          }
        >
          {isAnalyzing ? 'Analyzing...' : 'Analyze PCAP'}
        </Button>
      </CardContent>
    </Card>
  );
};

export default PcapUploader;
