import { useTranslation } from 'react-i18next';
import { CategoryChip } from '../../shared/CategoryChip';
import { Card, CardContent } from '../../ui/Card';
import { H2 } from '../../ui/Typography';
import { YamlEditor } from '../config/YamlEditor';

export interface YamlPreviewSectionProps {
  yamlContent: string;
}

export function YamlPreviewSection({ yamlContent }: YamlPreviewSectionProps) {
  const { t } = useTranslation('devices');
  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent>
        <H2 className="mb-heading flex items-center gap-compact text-base">
          <span>{t('editor.sections.yamlPreview.title')}</span>
          <CategoryChip label={t('editor.sections.yamlPreview.readOnly')} />
        </H2>
        <YamlEditor
          ariaLabel={t('editor.sections.yamlPreview.editorAria')}
          value={yamlContent}
          readOnly={true}
          height="auto"
          minHeight="150px"
          maxHeight="300px"
        />
      </CardContent>
    </Card>
  );
}
