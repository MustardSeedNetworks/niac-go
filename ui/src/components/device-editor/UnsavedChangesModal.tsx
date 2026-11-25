import { useTranslation } from 'react-i18next';
import { Button } from '../../shared/Button';
import { ActionButton } from '../../ui/ActionButton';
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
        <Button
          variant="secondary"
          data-testid="unsaved-cancel"
          disabled={saving}
          onClick={onCancel}
        >
          {t('buttons.cancel')}
        </Button>
        <Button
          variant="secondary"
          data-testid="unsaved-discard"
          disabled={saving}
          onClick={onDiscard}
        >
          {t('buttons.discard')}
        </Button>
        {onSave && (
          <ActionButton
            variant="primary"
            action="edit"
            data-testid="unsaved-save"
            disabled={saving}
            onClick={onSave}
          >
            {t('buttons.save')}
          </ActionButton>
        )}
      </div>
    </Modal>
  );
}
