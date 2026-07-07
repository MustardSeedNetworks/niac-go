import { valibotResolver } from '@hookform/resolvers/valibot';
import { Copy } from 'lucide-react';
import type { FC } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { type CloneDeviceFormFields, CloneDeviceSchema } from '../../schemas/forms';
import { Button } from '../../ui/Button';

interface CloneDeviceModalProps {
  hostname: string;
  onClone: (newHostname: string) => void;
  onCancel: () => void;
}

export const CloneDeviceModal: FC<CloneDeviceModalProps> = ({ hostname, onClone, onCancel }) => {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const {
    register,
    handleSubmit,
    formState: { errors, isValid },
  } = useForm<CloneDeviceFormFields>({
    resolver: valibotResolver(CloneDeviceSchema),
    defaultValues: { newHostname: `${hostname}-copy` },
    mode: 'onBlur',
  });

  const onSubmit: SubmitHandler<CloneDeviceFormFields> = ({ newHostname }) => {
    onClone(newHostname);
  };

  return (
    <div className="fixed inset-0 z-50 flex-center">
      <button
        type="button"
        className="absolute inset-0 bg-scrim/70 backdrop-blur-sm"
        onClick={onCancel}
        aria-label={t('list.cloneModal.closeAria')}
      />
      <div
        className="mx-4 w-full max-w-md rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl"
        role="dialog"
        aria-modal="true"
      >
        <form onSubmit={handleSubmit(onSubmit)} className="pad-lg stack-lg">
          <div className="flex items-center gap-default text-status-info">
            <Copy className={iconSizes.xl} />
            <h2 className="heading-3">{t('list.cloneModal.title')}</h2>
          </div>
          <p className="text-text-secondary">
            {t('list.cloneModal.descriptionPrefix')} <strong>{hostname}</strong>{' '}
            {t('list.cloneModal.descriptionSuffix')}
          </p>
          <div>
            <label
              htmlFor="new-hostname"
              className="block text-sm font-medium text-text-secondary mb-2"
            >
              {t('list.cloneModal.newHostnameLabel')}
            </label>
            <input
              id="new-hostname"
              type="text"
              {...register('newHostname')}
              className="w-full rounded-lg border border-surface-border bg-bg-base/60 pad-sm text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
            />
            {errors.newHostname ? (
              <p className="mt-inline text-xs text-status-error">{errors.newHostname.message}</p>
            ) : null}
          </div>
          <div className="flex justify-end gap-default pt-2">
            <Button variant="outline" type="button" onClick={onCancel}>
              {tCommon('buttons.cancel')}
            </Button>
            <Button tone="violet" type="submit" disabled={!isValid}>
              {tCommon('buttons.clone')}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
