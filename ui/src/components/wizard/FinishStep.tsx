import { PartyPopper, Save } from 'lucide-react';
import { type FC, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';
import { fetchConfig, updateConfig } from '../../api/client';
import { iconSizes } from '../../constants/sizes';
import { useErrorToast } from '../../hooks/useErrorToast';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, SmallText } from '../../ui/Typography';

/**
 * Step 5 — Save / Start. The simulation is already running by this point
 * (step 1 had to start it against the picked config so the existing
 * device-editing surface in steps 2-3 had something live to operate on —
 * there's no separate "load config without starting" endpoint). This step
 * re-saves the final config via the same `PUT /api/v1/config` the Devices
 * page's YAML editor uses, then hands off to the existing Simulation page,
 * which already shows the running status.
 */
export const FinishStep: FC = () => {
  const { t } = useTranslation('pages');
  const showError = useErrorToast();
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      const current = await fetchConfig();
      await updateConfig({ content: current.content });
      setSaved(true);
    } catch (err) {
      showError(err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex items-center gap-default">
          <PartyPopper className={`${iconSizes.lg} text-brand-accent`} />
          <H2>{t('newSimWizard.finish.title')}</H2>
        </div>
        <SmallText className="text-text-muted">{t('newSimWizard.finish.help')}</SmallText>

        <div className="flex flex-wrap gap-default">
          <Button
            tone="violet"
            data-testid="wizard-save-button"
            leftIcon={<Save className={iconSizes.md} />}
            disabled={saving}
            onClick={handleSave}
          >
            {saving ? t('newSimWizard.finish.savingLabel') : t('newSimWizard.finish.saveLabel')}
          </Button>
          <Link to="/runtime">
            <Button variant="outline" data-testid="wizard-goto-runtime">
              {t('newSimWizard.finish.gotoRuntime')}
            </Button>
          </Link>
        </div>

        {saved && (
          <SmallText className="text-status-success" role="status" aria-live="polite">
            {t('newSimWizard.finish.saved')}
          </SmallText>
        )}
      </CardContent>
    </Card>
  );
};
