import { Building2, FileCode, Globe, Layers, Router, Server, Shield, Wifi } from 'lucide-react';
import { type FC, memo } from 'react';
import type { Template } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';

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
  firewall: Shield,
  complete: Building2,
  custom: FileCode,
};

const typeColors: Record<Template['type'], string> = {
  basic: 'bg-status-info/20 text-status-info border-status-info/30',
  router: 'bg-status-warning/20 text-status-warning border-status-warning/30',
  switch: 'bg-status-success/20 text-status-success border-status-success/30',
  'access-point': 'bg-brand-primary/20 text-brand-accent border-brand-primary/30',
  server: 'bg-status-info/20 text-status-info border-status-info/30',
  firewall: 'bg-status-error/20 text-status-error border-status-error/30',
  complete: 'bg-status-warning/20 text-status-warning border-status-warning/30',
  custom: 'bg-pink-500/20 text-pink-300 border-pink-500/30',
};

export const TemplateCard: FC<TemplateCardProps> = memo(({ template, onView, onUse }) => {
  const IconComponent = typeIcons[template.type] || FileCode;
  const colorClass = typeColors[template.type] || typeColors.custom;

  return (
    <Card className="border-surface-border bg-bg-surface/70 hover:border-brand-primary/30 transition-colors">
      <CardContent className="space-y-4">
        {/* Header with icon and type */}
        <div className="flex items-start justify-between">
          <div className={`rounded-lg p-3 border ${colorClass}`}>
            <IconComponent className={iconSizes.xl} />
          </div>
          <Tag colorScheme="gray" className="text-xs capitalize">
            {template.type}
          </Tag>
        </div>

        {/* Template info */}
        <div className="space-y-1">
          <h3 className="font-semibold text-text-primary text-lg">
            {template.displayName || template.name}
          </h3>
          {template.displayName && (
            <SmallText className="text-text-muted font-mono text-[10px]">{template.name}</SmallText>
          )}
          <SmallText className="text-text-muted line-clamp-2">
            {template.description || 'No description available'}
          </SmallText>
        </div>

        {/* Device count and tags */}
        <div className="flex items-center gap-2 flex-wrap">
          <Tag colorScheme="purple">
            {template.deviceCount} {template.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
          {template.tags?.slice(0, 2).map((tag) => (
            <Tag key={tag} colorScheme="gray" className="text-xs">
              {tag}
            </Tag>
          ))}
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 pt-2">
          <Button tone="violet" size="sm" className="flex-1" onClick={() => onUse(template)}>
            Use
          </Button>
          <Button variant="outline" size="sm" className="flex-1" onClick={() => onView(template)}>
            View
          </Button>
        </div>
      </CardContent>
    </Card>
  );
});

TemplateCard.displayName = 'TemplateCard';
