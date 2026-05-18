import { AlertCircle, FileCode, Search, Upload } from 'lucide-react';
import type { FC } from 'react';
import { iconSizes } from '../../constants/sizes';
import { Button } from '../../ui/Button';
import { Card, CardContent } from '../../ui/Card';
import { H2, P, SmallText } from '../../ui/Typography';

interface LoadingStateProps {
  show: boolean;
}

export const TemplatesLoadingState: FC<LoadingStateProps> = ({ show }) => {
  if (!show) {
    return null;
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="flex items-center justify-center py-12">
        <div className="flex items-center gap-3 text-text-muted">
          <div
            className={`${iconSizes.lg} animate-spin rounded-full border-2 border-brand-primary border-t-transparent`}
          />
          <span>Loading templates...</span>
        </div>
      </CardContent>
    </Card>
  );
};

interface ErrorStateProps {
  error: Error | null;
  onRetry: () => void;
}

export const TemplatesErrorState: FC<ErrorStateProps> = ({ error, onRetry }) => {
  if (!error) {
    return null;
  }

  return (
    <Card className="border-status-error/30 bg-status-error/20">
      <CardContent className="space-y-3">
        <div className="flex items-start gap-3">
          <AlertCircle className={`mt-1 ${iconSizes.lg} text-status-error`} />
          <div>
            <p className="font-semibold text-status-error">Failed to Load Templates</p>
            <SmallText className="text-status-error/90">{error.message}</SmallText>
            <Button variant="outline" size="sm" className="mt-3" onClick={onRetry}>
              Retry
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

interface EmptyStateProps {
  show: boolean;
  onUploadClick: () => void;
}

export const TemplatesEmptyState: FC<EmptyStateProps> = ({ show, onUploadClick }) => {
  if (!show) {
    return null;
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="py-12 text-center">
        <FileCode className={`mx-auto ${iconSizes['3xl']} text-text-disabled`} />
        <H2 className="mt-4 mb-2">No Templates Available</H2>
        <P className="text-text-muted">Upload your first configuration template to get started.</P>
        <Button
          tone="violet"
          className="mt-4"
          leftIcon={<Upload className={iconSizes.md} />}
          onClick={onUploadClick}
        >
          Upload Template
        </Button>
      </CardContent>
    </Card>
  );
};

interface NoResultsStateProps {
  show: boolean;
  searchQuery: string;
  onClearSearch: () => void;
}

export const TemplatesNoResultsState: FC<NoResultsStateProps> = ({
  show,
  searchQuery,
  onClearSearch,
}) => {
  if (!show) {
    return null;
  }

  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="py-12 text-center">
        <Search className={`mx-auto ${iconSizes['3xl']} text-text-disabled`} />
        <H2 className="mt-4 mb-2">No Matching Templates</H2>
        <P className="text-text-muted">
          No templates match your search &quot;{searchQuery}&quot;. Try a different search term.
        </P>
        <Button variant="outline" className="mt-4" onClick={onClearSearch}>
          Clear Search
        </Button>
      </CardContent>
    </Card>
  );
};
