import { AlertCircle, FileCode, Search, Upload, X } from 'lucide-react';
import type { FC } from 'react';
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
    <Card className="border-white/5 bg-gray-900/70">
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <H2 className="mb-0 flex items-center gap-2">
              <FileCode className="h-5 w-5 text-violet-300" />
              Configuration Templates
            </H2>
            <P className="text-gray-400 mt-1">
              Browse and use pre-configured network templates to quickly start simulations.
            </P>
          </div>
          <Button tone="violet" leftIcon={<Upload className="h-4 w-4" />} onClick={onUploadClick}>
            Upload Template
          </Button>
        </div>

        {/* Search bar */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search templates by name, description, or type..."
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="w-full rounded-lg border border-white/10 bg-gray-950/60 py-3 pl-10 pr-10 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
          />
          {searchQuery && (
            <button
              type="button"
              onClick={() => onSearchChange('')}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
              aria-label="Clear search"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>

        {/* Status message */}
        {message && (
          <div
            className={`flex items-center gap-2 rounded-lg p-3 ${
              message.type === 'success'
                ? 'border border-green-500/30 bg-green-500/10 text-green-300'
                : 'border border-red-500/30 bg-red-500/10 text-red-300'
            }`}
            role="alert"
          >
            {message.type === 'error' && <AlertCircle className="h-4 w-4" />}
            <span>{message.text}</span>
            <button
              type="button"
              onClick={onDismissMessage}
              className="ml-auto text-current hover:opacity-70"
              aria-label="Dismiss message"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
