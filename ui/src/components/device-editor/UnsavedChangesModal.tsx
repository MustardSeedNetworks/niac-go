import { useTranslation } from 'react-i18next';
import { Button } from '../../ui/Button';
import { Modal } from '../../ui/Modal';
import { SmallText } from '../../ui/Typography';

interface Props {
  open: boolean;
  saving: boolean;
  onSave?: () => void;
  onDiscard: () => void;
  onCancel: () => void;
  error?: string;
}

export function UnsavedChangesModal({ open, saving, onSave, onDiscard, onCancel, error }: Props) {
  const { t } = useTranslation('common');
  return (
    <Modal
      isOpen={open}
      title={t('unsaved.title')}
      onClose={onCancel}
      closeOnEscape={!saving}
      closeOnBackdropClick={!saving}
      showCloseButton={false}
      size="sm"
    >
      <SmallText>{onSave ? t('unsaved.message') : t('unsaved.discardMessage')}</SmallText>
      {error && (
        <SmallText role="alert" className="text-status-error">
          {error}
        </SmallText>
      )}
      <div className="flex justify-end gap-default mt-heading">
        <Button data-testid="unsaved-cancel" variant="outline" disabled={saving} onClick={onCancel}>
          {t('buttons.cancel')}
        </Button>
        <Button
          data-testid="unsaved-discard"
          variant="outline"
          disabled={saving}
          onClick={onDiscard}
        >
          {t('buttons.discard')}
        </Button>
        {onSave && (
          <Button data-testid="unsaved-save" disabled={saving} onClick={onSave}>
            {t('buttons.save')}
          </Button>
        )}
      </div>
    </Modal>
  );
}
