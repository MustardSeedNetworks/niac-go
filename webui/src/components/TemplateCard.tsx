import { memo, type FC } from 'react';
import { Globe, Router, Layers, Wifi, Server, Building2, FileCode } from 'lucide-react';
import { Card, CardContent, Button, Tag, SmallText } from '@krisarmstrong/web-foundation';
import type { Template } from '../api/types';

interface TemplateCardProps {
  template: Template;
  onView: (template: Template) => void;
  onUse: (template: Template) => void;
}

const typeIcons: Record<Template['type'], FC<{ className?: string }>> = {
  basic: Globe,
  router: Router,
  switch: Layers,
  'access-point': Wifi,
  server: Server,
  complete: Building2,
  custom: FileCode,
};

const typeColors: Record<Template['type'], string> = {
  basic: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
  router: 'bg-orange-500/20 text-orange-300 border-orange-500/30',
  switch: 'bg-green-500/20 text-green-300 border-green-500/30',
  'access-point': 'bg-purple-500/20 text-purple-300 border-purple-500/30',
  server: 'bg-cyan-500/20 text-cyan-300 border-cyan-500/30',
  complete: 'bg-amber-500/20 text-amber-300 border-amber-500/30',
  custom: 'bg-pink-500/20 text-pink-300 border-pink-500/30',
};

export const TemplateCard: FC<TemplateCardProps> = memo(({ template, onView, onUse }) => {
  const IconComponent = typeIcons[template.type] || FileCode;
  const colorClass = typeColors[template.type] || typeColors.custom;

  return (
    <Card className="border-white/5 bg-gray-900/70 hover:border-violet-500/30 transition-colors">
      <CardContent className="space-y-4">
        {/* Header with icon and type */}
        <div className="flex items-start justify-between">
          <div className={`rounded-lg p-3 border ${colorClass}`}>
            <IconComponent className="h-6 w-6" />
          </div>
          <Tag colorScheme="gray" className="text-xs capitalize">
            {template.type}
          </Tag>
        </div>

        {/* Template info */}
        <div className="space-y-1">
          <h3 className="font-semibold text-white text-lg">{template.name}</h3>
          <SmallText className="text-gray-400 line-clamp-2">
            {template.description || 'No description available'}
          </SmallText>
        </div>

        {/* Device count and tags */}
        <div className="flex items-center gap-2 flex-wrap">
          <Tag colorScheme="purple">
            {template.device_count} {template.device_count === 1 ? 'device' : 'devices'}
          </Tag>
          {template.tags?.slice(0, 2).map((tag) => (
            <Tag key={tag} colorScheme="gray" className="text-xs">
              {tag}
            </Tag>
          ))}
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 pt-2">
          <Button
            tone="violet"
            size="sm"
            className="flex-1"
            onClick={() => onUse(template)}
          >
            Use
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="flex-1"
            onClick={() => onView(template)}
          >
            View
          </Button>
        </div>
      </CardContent>
    </Card>
  );
});

TemplateCard.displayName = 'TemplateCard';
