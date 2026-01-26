import type { FC } from 'react';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { H2 } from '../../ui/Typography';
import { YamlEditor } from '../config/YamlEditor';

export interface YamlPreviewSectionProps {
  yamlContent: string;
}

export const YamlPreviewSection: FC<YamlPreviewSectionProps> = ({ yamlContent }) => {
  return (
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent>
        <H2 className="mb-3 flex items-center gap-2 text-base">
          <span>YAML Preview</span>
          <Tag colorScheme="gray">Read-only</Tag>
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
