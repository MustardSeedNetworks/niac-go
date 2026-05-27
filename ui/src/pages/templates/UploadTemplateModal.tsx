import { zodResolver } from '@hookform/resolvers/zod';
import { Upload, X } from 'lucide-react';
import { type FC, useState } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { uploadTemplate } from '../../api/client';
import { iconSizes } from '../../constants/sizes';
import { type UploadTemplateFormFields, UploadTemplateSchema } from '../../schemas/forms';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';

interface UploadTemplateModalProps {
  onClose: () => void;
  onSuccess: () => void;
  onError: (error: Error) => void;
}

export const UploadTemplateModal: FC<UploadTemplateModalProps> = ({
  onClose,
  onSuccess,
  onError,
}) => {
  const { t } = useTranslation('pages');
  const [uploadFile, setUploadFile] = useState<File | null>(null);

  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<UploadTemplateFormFields>({
    resolver: zodResolver(UploadTemplateSchema),
    defaultValues: { name: '', description: '', content: '', type: 'custom' },
    mode: 'onBlur',
  });

  const name = watch('name');

  const handleFileChange = async (file: File | null) => {
    if (!file) {
      setUploadFile(null);
      return;
    }

    // Validate file size (1MB limit for templates)
    const MaxSize = 1024 * 1024;
    if (file.size > MaxSize) {
      onError(new Error(t('templates.uploadModal.errorFileTooLarge')));
      return;
    }

    if (!file.name.match(/\.(yaml|yml)$/i)) {
      onError(new Error(t('templates.uploadModal.errorInvalidFile')));
      return;
    }

    setUploadFile(file);

    try {
      const text = await file.text();
      setValue('content', text, { shouldValidate: true, shouldDirty: true });
      // Auto-fill name from filename if empty
      if (!name) {
        setValue('name', file.name.replace(/\.(yaml|yml)$/i, ''), {
          shouldValidate: true,
          shouldDirty: true,
        });
      }
    } catch {
      onError(new Error(t('templates.uploadModal.errorReadFailed')));
    }
  };

  const onSubmit: SubmitHandler<UploadTemplateFormFields> = async (values) => {
    try {
      await uploadTemplate({
        name: values.name,
        description: values.description,
        content: values.content,
        type: values.type,
      });
      onSuccess();
    } catch (err) {
      onError(err as Error);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onClose}
        aria-label={t('templates.uploadModal.backdropAriaLabel')}
      />
      <div
        className="relative mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="upload-modal-title"
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-surface-border px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-brand-primary/20 p-2">
              <Upload className={`${iconSizes.lg} text-brand-accent`} />
            </div>
            <div>
              <h2 id="upload-modal-title" className="text-lg font-semibold text-text-primary">
                {t('templates.uploadModal.title')}
              </h2>
              <SmallText className="text-text-muted">
                {t('templates.uploadModal.subtitle')}
              </SmallText>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-text-muted hover:bg-surface-hover hover:text-text-primary transition-colors"
            aria-label={t('templates.uploadModal.closeAriaLabel')}
          >
            <X className={iconSizes.lg} />
          </button>
        </div>

        <form onSubmit={handleSubmit(onSubmit)}>
          {/* Modal Content */}
          <div className="space-y-4 p-6">
            {/* File upload */}
            <div>
              <label
                htmlFor="template-upload"
                className="mb-2 block text-sm font-medium text-text-secondary"
              >
                {t('templates.uploadModal.fileLabel')}
              </label>
              <input
                id="template-upload"
                type="file"
                accept=".yaml,.yml"
                onChange={(e) => handleFileChange(e.target.files?.[0] || null)}
                className="w-full cursor-pointer rounded-lg border border-dashed border-surface-border bg-bg-base/40 p-3 text-sm text-text-primary file:mr-3 file:rounded-md file:border-0 file:bg-brand-primary file:px-3 file:py-1 file:text-sm file:font-medium"
              />
              {uploadFile && (
                <SmallText className="mt-1 text-status-success">
                  {t('templates.uploadModal.fileSelected', { filename: uploadFile.name })}
                </SmallText>
              )}
            </div>

            {/* Name input */}
            <div>
              <label
                htmlFor="template-name"
                className="mb-2 block text-sm font-medium text-text-secondary"
              >
                {t('templates.uploadModal.nameLabel')}
              </label>
              <input
                id="template-name"
                type="text"
                {...register('name')}
                placeholder={t('templates.uploadModal.namePlaceholder')}
                className="w-full rounded-lg border border-surface-border bg-bg-base/60 p-3 text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none"
              />
              {errors.name ? (
                <p className="mt-1 text-xs text-status-error">{errors.name.message}</p>
              ) : null}
            </div>

            {/* Description input */}
            <div>
              <label
                htmlFor="template-description"
                className="mb-2 block text-sm font-medium text-text-secondary"
              >
                {t('templates.uploadModal.descriptionLabel')}
              </label>
              <textarea
                id="template-description"
                {...register('description')}
                placeholder={t('templates.uploadModal.descriptionPlaceholder')}
                rows={2}
                className="w-full rounded-lg border border-surface-border bg-bg-base/60 p-3 text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none resize-none"
              />
              {errors.description ? (
                <p className="mt-1 text-xs text-status-error">{errors.description.message}</p>
              ) : null}
            </div>

            {/* Type selector */}
            <div>
              <label
                htmlFor="template-type"
                className="mb-2 block text-sm font-medium text-text-secondary"
              >
                {t('templates.uploadModal.typeLabel')}
              </label>
              <select
                id="template-type"
                {...register('type')}
                className="w-full rounded-lg border border-surface-border bg-bg-base/60 p-3 text-sm text-text-primary focus:border-brand-accent focus:outline-none"
              >
                <option value="basic">{t('templates.uploadModal.categoryBasic')}</option>
                <option value="router">{t('templates.uploadModal.categoryRouter')}</option>
                <option value="switch">{t('templates.uploadModal.categorySwitch')}</option>
                <option value="access-point">
                  {t('templates.uploadModal.categoryAccessPoint')}
                </option>
                <option value="server">{t('templates.uploadModal.categoryServer')}</option>
                <option value="complete">{t('templates.uploadModal.categoryComplete')}</option>
                <option value="custom">{t('templates.uploadModal.categoryCustom')}</option>
              </select>
            </div>

            {/* Content textarea */}
            <div>
              <label
                htmlFor="template-content"
                className="mb-2 block text-sm font-medium text-text-secondary"
              >
                {t('templates.uploadModal.contentLabel')}
              </label>
              <textarea
                id="template-content"
                {...register('content')}
                placeholder={t('templates.uploadModal.contentPlaceholder')}
                rows={10}
                className="w-full rounded-lg border border-surface-border bg-bg-base/60 p-3 font-mono text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none resize-none"
                spellCheck={false}
              />
              {errors.content ? (
                <p className="mt-1 text-xs text-status-error">{errors.content.message}</p>
              ) : null}
            </div>
          </div>

          {/* Modal Footer */}
          <div className="flex justify-end gap-3 border-t border-surface-border px-6 py-4 bg-bg-base/50">
            <Button variant="outline" type="button" onClick={onClose} disabled={isSubmitting}>
              {t('templates.uploadModal.cancel')}
            </Button>
            <Button tone="violet" type="submit" disabled={isSubmitting}>
              {isSubmitting
                ? t('templates.uploadModal.uploading')
                : t('templates.uploadModal.uploadButton')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
