import { FileCog, Save } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { SmallText } from '../../ui/Typography';
import { YamlEditor } from '../config/YamlEditor';

interface DevicesStepProps {
  draftName: string;
  content: string;
  dirty: boolean;
  saving: boolean;
  onChange: (content: string) => void;
  onSave: () => void;
}

/**
 * Step 2 edits the saved draft directly. The visual device composer replaces
 * this YAML surface in Phase 3; until then, server validation and revision
 * matching keep this editor safe without touching the active runtime.
 */
export const DevicesStep: FC<DevicesStepProps> = ({
  draftName,
  content,
  dirty,
  saving,
  onChange,
  onSave,
}) => {
  const { t } = useTranslation('pages');
  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="stack">
        <div className="flex items-center gap-default">
          <FileCog className={`${iconSizes.lg} text-brand-accent`} />
          <div>
            <p className="font-medium text-text-primary">{draftName}</p>
            <SmallText className="text-text-muted">{t('newSimWizard.devices.help')}</SmallText>
          </div>
        </div>
        <YamlEditor
          value={content}
          onChange={onChange}
          readOnly={saving}
          height="24rem"
          className="mt-heading"
        />
        <div>
          <Button
            tone="violet"
            leftIcon={<Save className={iconSizes.md} />}
            disabled={!dirty || saving}
            loading={saving}
            onClick={onSave}
          >
            {saving ? t('newSimWizard.devices.savingLabel') : t('newSimWizard.devices.saveLabel')}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};
