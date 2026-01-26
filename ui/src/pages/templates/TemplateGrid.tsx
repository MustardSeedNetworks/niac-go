import type { FC } from 'react';
import type { Template } from '../../api/types';
import { TemplateCard } from '../../components/TemplateCard';
import { Tag } from '../../ui/Tag';
import { H2 } from '../../ui/Typography';

interface TemplateGridProps {
  templatesByType: Record<string, Template[]>;
  onView: (template: Template) => void;
  onUse: (template: Template) => void;
}

export const TemplateGrid: FC<TemplateGridProps> = ({ templatesByType, onView, onUse }) => {
  const entries = Object.entries(templatesByType);

  if (entries.length === 0) {
    return null;
  }

  return (
    <div className="space-y-8">
      {entries.map(([type, typeTemplates]) => (
        <div key={type} className="space-y-4">
          <div className="flex items-center gap-3">
            <H2 className="mb-0 capitalize">{type} Templates</H2>
            <Tag colorScheme="gray">{typeTemplates.length}</Tag>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {typeTemplates.map((template) => (
              <TemplateCard key={template.name} template={template} onView={onView} onUse={onUse} />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};
