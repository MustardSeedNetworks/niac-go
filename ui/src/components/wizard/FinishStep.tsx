import { PartyPopper } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, SmallText } from '../../ui/Typography';

export const FinishStep: FC<{ draftName: string; sessionId?: string | null }> = ({
  draftName,
  sessionId,
}) => {
  const { t } = useTranslation('pages');

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex items-center gap-default">
          <PartyPopper className={`${iconSizes.lg} text-brand-accent`} />
          <H2>{t('newSimWizard.finish.title')}</H2>
        </div>
        <SmallText className="text-text-muted">{t('newSimWizard.finish.help')}</SmallText>
        {/* The session is what the author just started; the draft is only
            where it came from. Naming the session is what lets them find it
            among the others on the runtime page. */}
        {sessionId && (
          <SmallText data-testid="wizard-finish-session-id" className="font-mono text-text-primary">
            {t('newSimWizard.finish.sessionLabel', { id: sessionId })}
          </SmallText>
        )}
        <SmallText data-testid="wizard-finish-draft-name" className="font-mono text-text-muted">
          {t('newSimWizard.finish.draftLabel', { name: draftName })}
        </SmallText>

        <div>
          <Link to="/runtime">
            <Button tone="violet" data-testid="wizard-goto-runtime">
              {t('newSimWizard.finish.gotoRuntime')}
            </Button>
          </Link>
        </div>
      </CardContent>
    </Card>
  );
};
