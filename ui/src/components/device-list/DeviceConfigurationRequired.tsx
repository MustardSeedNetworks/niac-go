import { useTranslation } from 'react-i18next';
import { Card } from '../../shared/Card';
import { EmptyState } from '../../shared/EmptyState';
import { LinkButton } from '../../ui/LinkButton';

export function DeviceConfigurationRequired() {
  const { t } = useTranslation('devices');
  return (
    <Card>
      <EmptyState
        title={t('list.states.configurationRequiredTitle')}
        hint={t('list.states.configurationRequiredDescription')}
        action={
          <LinkButton to="/new-simulation" data-testid="create-device-draft">
            {t('list.states.createDraft')}
          </LinkButton>
        }
      />
    </Card>
  );
}
