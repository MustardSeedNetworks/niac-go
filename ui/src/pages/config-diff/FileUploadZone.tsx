import { AlertCircle, FileCode, Upload, X } from 'lucide-react';
import { type FC, useCallback, useState } from 'react';
import { iconSizes } from '../../constants/sizes';
import { SmallText } from '../../ui/Typography';
import { formatBytes } from '../../utils/format';

/**
 * UploadedFile is the validated representation of a YAML config the
 * user has dropped into one of the two upload slots on the Config Diff
 * page. Shared with ConfigDiffPage via re-export.
 */
export interface UploadedFile {
  name: string;
  content: string;
  size: number;
}

interface FileUploadZoneProps {
  label: string;
  file: UploadedFile | null;
  onFileUpload: (file: UploadedFile) => void;
  onClear: () => void;
  disabled?: boolean;
}

const MAX_FILE_SIZE = 1024 * 1024; // 1 MB

/**
 * FileUploadZone is the drag-and-drop YAML dropzone used twice on the
 * Config Diff page (once for each side of the comparison). When a file
 * is already loaded it switches into a compact "file loaded" tile with
 * a clear button instead.
 */
export const FileUploadZone: FC<FileUploadZoneProps> = ({
  label,
  file,
  onFileUpload,
  onClear,
  disabled,
}) => {
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

      // Validate file size
      if (uploadedFile.size > MAX_FILE_SIZE) {
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
      <div className="rounded-xl border border-status-success/30 bg-status-success/10 p-4">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-status-success/20 p-2">
              <FileCode className={`${iconSizes.lg} text-status-success`} />
            </div>
            <div>
              <p className="font-semibold text-text-primary">{file.name}</p>
              <SmallText className="text-status-success">
                {formatBytes(file.size)} | {file.content.split('\n').length} lines
              </SmallText>
            </div>
          </div>
          <button
            type="button"
            onClick={onClear}
            disabled={disabled}
            className="rounded-lg p-1.5 text-text-muted hover:bg-surface-hover hover:text-text-primary transition-colors disabled:opacity-50"
            aria-label={`Remove ${file.name}`}
          >
            <X className={iconSizes.md} />
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
            ? 'border-brand-accent bg-brand-primary/10'
            : 'border-surface-border hover:border-brand-accent/50 hover:bg-bg-base/50'
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
        <Upload className={`mx-auto ${iconSizes['2xl']} text-text-muted mb-2`} />
        <p className="text-text-secondary font-medium">{label}</p>
        <SmallText className="text-text-muted">
          Drag & drop or click to select a YAML file
        </SmallText>
      </label>
      {error && (
        <div className="flex items-center gap-2 text-status-error text-sm">
          <AlertCircle className={iconSizes.md} />
          {error}
        </div>
      )}
    </div>
  );
};
