import { useTranslation } from 'react-i18next';
import { ApiError } from '../../api/errors';
import { Card, CardContent } from '../../ui/Card';

export function WizardStatusNotice({ loading, error }: { loading: boolean; error: Error | null }) {
  const { t } = useTranslation('pages');
  const wrongMode = error instanceof ApiError && error.status === 501;
  const unavailable = !error || (error instanceof ApiError && error.status === 0);
  return (
    <Card>
      <CardContent>
        <div
          data-testid="wizard-status-notice"
          role={loading ? 'status' : 'alert'}
          className="stack"
        >
          <p>
            {loading
              ? t('newSimWizard.status.loading')
              : wrongMode
                ? t('newSimWizard.status.wrongMode')
                : unavailable
                  ? t('newSimWizard.status.unavailable')
                  : t('newSimWizard.status.failed')}
          </p>
          {wrongMode && <code>niac daemon</code>}
          {!loading && error && !wrongMode && <p>{error.message}</p>}
        </div>
      </CardContent>
    </Card>
  );
}
