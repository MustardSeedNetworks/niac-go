import { valibotResolver } from '@hookform/resolvers/valibot';
import { Copy } from 'lucide-react';
import { type FC, useId } from 'react';
import { type SubmitHandler, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { type CloneDeviceFormFields, CloneDeviceSchema } from '../../schemas/forms';
import { Button } from '../../ui/Button';
import { Modal } from '../../ui/Modal';

interface CloneDeviceModalProps {
  hostname: string;
  onClone: (newHostname: string) => void;
  onCancel: () => void;
}

export const CloneDeviceModal: FC<CloneDeviceModalProps> = ({ hostname, onClone, onCancel }) => {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const titleId = useId();
  // The submit button sits in the dialog footer, outside <form>, so it needs
  // the form's id to submit it.
  const formId = useId();
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
    <Modal
      isOpen
      onClose={onCancel}
      size="md"
      showCloseButton={false}
      labelledBy={titleId}
      footer={
        <>
          <Button variant="outline" type="button" onClick={onCancel}>
            {tCommon('buttons.cancel')}
          </Button>
          <Button tone="violet" type="submit" form={formId} disabled={!isValid}>
            {tCommon('buttons.clone')}
          </Button>
        </>
      }
    >
      <form id={formId} onSubmit={handleSubmit(onSubmit)} className="stack-lg">
        <div className="flex items-center gap-default text-status-info">
          <Copy className={iconSizes.xl} />
          <h2 id={titleId} className="heading-3">
            {t('list.cloneModal.title')}
          </h2>
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
      </form>
    </Modal>
  );
};
