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
      <div className="rounded-xl border border-green-500/30 bg-green-500/10 p-4">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-green-500/20 p-2">
              <FileCode className={`${iconSizes.lg} text-green-300`} />
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
        <Upload className={`mx-auto ${iconSizes['2xl']} text-gray-400 mb-2`} />
        <p className="text-gray-300 font-medium">{label}</p>
        <SmallText className="text-gray-500">Drag & drop or click to select a YAML file</SmallText>
      </label>
      {error && (
        <div className="flex items-center gap-2 text-red-400 text-sm">
          <AlertCircle className={iconSizes.md} />
          {error}
        </div>
      )}
    </div>
  );
};
