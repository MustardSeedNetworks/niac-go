import { useTranslation } from 'react-i18next';
import { Alert } from '../../shared/Alert';

export interface StatusMessage {
  type: 'success' | 'error';
  text: string;
}

interface DeviceStatusMessageProps {
  message: StatusMessage | null;
  onDismiss: () => void;
}

export function DeviceStatusMessage({ message, onDismiss }: DeviceStatusMessageProps) {
  const { t } = useTranslation('common');
  if (!message) {
    return null;
  }
  return (
    <Alert
      tone={message.type}
      onDismiss={onDismiss}
      dismissLabel={t('accessibility.dismissMessage')}
    >
      {message.text}
    </Alert>
  );
}
