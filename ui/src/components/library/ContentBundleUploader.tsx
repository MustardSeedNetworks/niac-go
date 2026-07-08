import { AlertCircle, FileArchive, PackageCheck, Upload, X } from 'lucide-react';
import { type FC, useCallback, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { isApiError } from '../../api/errors';
import { installContentBundleWithProgress } from '../../api/library-client';
import { iconSizes } from '../../constants/sizes';
import { useErrorToast } from '../../hooks/useErrorToast';
import { useUIStore } from '../../stores/ui-store';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { fileToBase64 } from '../../utils/file';
import { formatBytes } from '../../utils/format';

/**
 * ContentBundleUploader — the UI upload path for a content bundle
 * (gzip-tar mirroring the library layout: networks/, walks/, pcaps/),
 * i.e. the air-gapped/manual install counterpart to
 * `niac content install --bundle` (#897 L3b). POSTs to
 * /api/v1/library/install, which is admin-scoped like config/import — the
 * caller is expected to gate this behind <RequireScope min="admin">.
 *
 * Self-contained (owns its own upload state) rather than split into a
 * presentational component + page-level orchestration like the PCAP
 * uploader: there's no follow-on analysis view here, just "pick a file,
 * install it, see the result."
 */

/** Advertised max: mirrors internal/api's MaxLibraryBundleUploadSize (512MB
 * raw gzip-tar). The server enforces its own hard cap independently. */
const MAX_BUNDLE_SIZE = 512 * 1024 * 1024;

const ACCEPTED_EXTENSIONS = ['.tar.gz', '.tgz'];

function validateFile(file: File, invalidTypeMsg: string, tooLargeMsg: string, emptyMsg: string) {
  const name = file.name.toLowerCase();
  if (!ACCEPTED_EXTENSIONS.some((ext) => name.endsWith(ext))) {
    return invalidTypeMsg;
  }
  if (file.size > MAX_BUNDLE_SIZE) {
    return tooLargeMsg;
  }
  if (file.size === 0) {
    return emptyMsg;
  }
  return null;
}

export const ContentBundleUploader: FC = () => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const showError = useErrorToast();
  const addNotification = useUIStore((s) => s.addNotification);

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [installing, setInstalling] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) {
        return;
      }
      const error = validateFile(
        file,
        t('libraryFiles.installBundle.invalidFileType', {
          extensions: ACCEPTED_EXTENSIONS.join(', '),
        }),
        t('libraryFiles.installBundle.fileTooLarge', { size: formatBytes(MAX_BUNDLE_SIZE) }),
        t('libraryFiles.installBundle.fileEmpty'),
      );
      if (error) {
        if (fileInputRef.current) {
          fileInputRef.current.value = '';
        }
        setValidationError(error);
        setSelectedFile(null);
        return;
      }
      setValidationError(null);
      setSelectedFile(file);
    },
    [t],
  );

  const handleClear = useCallback(() => {
    setSelectedFile(null);
    setValidationError(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  }, []);

  const handleInstall = useCallback(async () => {
    if (!selectedFile) {
      return;
    }
    setInstalling(true);
    setValidationError(null);
    setUploadProgress(0);
    try {
      const data = await fileToBase64(selectedFile);
      const result = await installContentBundleWithProgress(
        { filename: selectedFile.name, data },
        setUploadProgress,
      );
      addNotification({
        type: 'success',
        title: t('libraryFiles.installBundle.installSuccessTitle'),
        message: t('libraryFiles.installBundle.installSuccessMessage', {
          networks: result.perKind.networks ?? 0,
          walks: result.perKind.walks ?? 0,
          pcaps: result.perKind.pcaps ?? 0,
        }),
      });
      handleClear();
    } catch (err) {
      if (isApiError(err) && err.code === 'request_too_large') {
        const limit =
          err.details.find((detail) => detail.field === 'body')?.value ??
          formatBytes(MAX_BUNDLE_SIZE);
        setValidationError(t('libraryFiles.installBundle.serverFileTooLarge', { size: limit }));
      } else {
        showError(err, t('libraryFiles.installBundle.installFailedTitle'));
      }
    } finally {
      setInstalling(false);
      setUploadProgress(null);
    }
  }, [selectedFile, addNotification, showError, t, handleClear]);

  return (
    <Card className="border-surface-border bg-bg-surface/70" data-testid="content-bundle-uploader">
      <CardContent className="stack-lg">
        <div>
          <h3 className="text-sm font-medium text-text-primary">
            {t('libraryFiles.installBundle.title')}
          </h3>
          <SmallText className="text-text-muted">
            {t('libraryFiles.installBundle.description')}
          </SmallText>
        </div>

        <input
          ref={fileInputRef}
          type="file"
          accept=".tar.gz,.tgz"
          onChange={handleFileChange}
          className="hidden"
          data-testid="content-bundle-file-input"
        />

        <div className="flex flex-wrap items-center gap-compact">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => fileInputRef.current?.click()}
            disabled={installing}
            leftIcon={<Upload className={iconSizes.md} />}
            data-testid="content-bundle-browse"
          >
            {t('libraryFiles.installBundle.browseButton')}
          </Button>

          {selectedFile && (
            <span className="inline-flex items-center gap-compact rounded-lg border border-surface-border bg-bg-base/50 px-cell py-1 text-xs text-text-primary">
              <FileArchive className={`${iconSizes.sm} text-brand-accent`} />
              {selectedFile.name} ({formatBytes(selectedFile.size)})
              <Button
                variant="ghost"
                size="xs"
                onClick={handleClear}
                disabled={installing}
                aria-label={tCommon('buttons.clear')}
              >
                <X className={iconSizes.sm} />
              </Button>
            </span>
          )}

          <Button
            tone="violet"
            size="sm"
            onClick={() => void handleInstall()}
            disabled={!selectedFile || installing}
            leftIcon={<PackageCheck className={iconSizes.md} />}
            data-testid="content-bundle-install"
          >
            {installing
              ? t('libraryFiles.installBundle.installingButton')
              : t('libraryFiles.installBundle.installButton')}
          </Button>
        </div>

        <SmallText className="text-text-muted">
          {t('libraryFiles.installBundle.maxFileSizeLabel', { size: formatBytes(MAX_BUNDLE_SIZE) })}
        </SmallText>

        {uploadProgress !== null && (
          <div className="stack-sm" data-testid="content-bundle-progress">
            <SmallText className="text-text-muted">
              {t('libraryFiles.installBundle.uploadingProgress', { percent: uploadProgress })}
            </SmallText>
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
                data-testid="content-bundle-progress-bar"
              />
            </div>
          </div>
        )}

        {validationError && (
          <div
            className="flex items-center gap-compact rounded-lg border border-status-error/30 bg-status-error/20 pad-sm"
            data-testid="content-bundle-error"
          >
            <AlertCircle className={`${iconSizes.lg} flex-shrink-0 text-status-error`} />
            <SmallText className="text-status-error">{validationError}</SmallText>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default ContentBundleUploader;
