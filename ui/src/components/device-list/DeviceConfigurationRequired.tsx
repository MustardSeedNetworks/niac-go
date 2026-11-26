import { useTranslation } from 'react-i18next';
import { ButtonLink } from '../../shared/Button';
import { Card } from '../../shared/Card';
import { EmptyState } from '../../shared/EmptyState';
import { linkTo } from '../../ui/linkTo';

export function DeviceConfigurationRequired() {
  const { t } = useTranslation('devices');
  return (
    <Card>
      <EmptyState
        title={t('list.states.configurationRequiredTitle')}
        hint={t('list.states.configurationRequiredDescription')}
        action={
          <ButtonLink
            variant="primary"
            renderLink={linkTo('/new-simulation', 'create-device-draft')}
          >
            {t('list.states.createDraft')}
          </ButtonLink>
        }
      />
    </Card>
  );
}
