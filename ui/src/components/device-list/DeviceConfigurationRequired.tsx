import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';
import { Card, CardContent } from '../../ui/Card';
import { H2, P } from '../../ui/Typography';

export function DeviceConfigurationRequired() {
  const { t } = useTranslation('devices');
  return (
    <Card>
      <CardContent className="stack-lg">
        <H2>{t('list.states.configurationRequiredTitle')}</H2>
        <P>{t('list.states.configurationRequiredDescription')}</P>
        <Link
          to="/new-simulation"
          data-testid="create-device-draft"
          className="inline-flex min-h-11 items-center self-start rounded-lg border border-brand-primary bg-brand-primary px-4 py-3 text-on-brand focus-visible:outline-2 focus-visible:outline-offset-2"
        >
          {t('list.states.createDraft')}
        </Link>
      </CardContent>
    </Card>
  );
}
