import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2 } from '../../ui/Typography';
import { YamlEditor } from '../config/YamlEditor';

export interface YamlPreviewSectionProps {
  yamlContent: string;
}

export const YamlPreviewSection: FC<YamlPreviewSectionProps> = ({ yamlContent }) => {
  const { t } = useTranslation('devices');
  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent>
        <H2 className="mb-heading flex items-center gap-compact text-base">
          <span>{t('editor.sections.yamlPreview.title')}</span>
          <Tag colorScheme="gray">{t('editor.sections.yamlPreview.readOnly')}</Tag>
        </H2>
        <YamlEditor
          value={yamlContent}
          readOnly={true}
          height="auto"
          minHeight="150px"
          maxHeight="300px"
        />
      </CardContent>
    </Card>
  );
};
