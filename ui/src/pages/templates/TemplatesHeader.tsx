import { AlertCircle, FileCode, Search, Upload, X } from 'lucide-react';
import type { FC } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, P } from '../../ui/Typography';
import type { StatusMessage } from './useTemplates';

interface TemplatesHeaderProps {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onUploadClick: () => void;
  message: StatusMessage | null;
  onDismissMessage: () => void;
}

export const TemplatesHeader: FC<TemplatesHeaderProps> = ({
  searchQuery,
  onSearchChange,
  onUploadClick,
  message,
  onDismissMessage,
}) => {
  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <H2 className="flex items-center gap-2">
              <FileCode className={`${iconSizes.lg} text-brand-accent`} />
              Configuration Templates
            </H2>
            <P className="text-text-muted mt-1">
              Browse and use pre-configured network templates to quickly start simulations.
            </P>
          </div>
          <Button
            tone="violet"
            leftIcon={<Upload className={iconSizes.md} />}
            onClick={onUploadClick}
          >
            Upload Template
          </Button>
        </div>

        {/* Search bar */}
        <div className="relative">
          <Search
            className={`absolute left-3 top-1/2 ${iconSizes.lg} -translate-y-1/2 text-text-muted`}
          />
          <input
            type="text"
            placeholder="Search templates by name, description, or type..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="w-full rounded-lg border border-surface-border bg-bg-base/60 py-3 pl-10 pr-10 text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => onSearchChange('')}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-primary"
              aria-label="Clear search"
            >
              <X className={iconSizes.md} />
            </button>
          )}
        </div>

        {/* Status message */}
        {message && (
          <div
            className={`flex items-center gap-2 rounded-lg p-3 ${
              message.type === 'success'
                ? 'border border-status-success/30 bg-status-success/10 text-status-success'
                : 'border border-status-error/30 bg-status-error/10 text-status-error'
            }`}
            role="alert"
          >
            {message.type === 'error' && <AlertCircle className={iconSizes.md} />}
            <span>{message.text}</span>
            <button
              type="button"
              onClick={onDismissMessage}
              className="ml-auto text-current hover:opacity-70"
              aria-label="Dismiss message"
            >
              <X className={iconSizes.md} />
            </button>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
