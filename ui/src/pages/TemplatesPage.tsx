import { AlertCircle, FileCode, Search, Upload, X } from 'lucide-react';
import { type FC, useCallback, useMemo, useState } from 'react';
import { applyTemplate, fetchTemplateContent, fetchTemplates } from '../api/client';
import type { Template, TemplateContent } from '../api/types';
import { TemplateCard } from '../components/TemplateCard';
import { TemplatePreviewModal } from '../components/TemplatePreviewModal';
import { UploadTemplateModal } from '../components/templates/UploadTemplateModal';
import { useApiResource } from '../hooks/useApiResource';
import { Button } from '../ui/Button';
import { Card, CardContent } from '../ui/Card';
import { InputModal } from '../ui/InputModal';
import { Tag } from '../ui/Tag';
import { H2, P, SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';

export const TemplatesPage: FC = () => {
  const {
    data: templates,
    loading: templatesLoading,
    error: templatesError,
    refetch: refetchTemplates,
  } = useApiResource(fetchTemplates, [], { intervalMs: 30000 });

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [templateContent, setTemplateContent] = useState<TemplateContent | null>(null);
  const [contentLoading, setContentLoading] = useState(false);
  const [contentError, setContentError] = useState<Error | null>(null);
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [message, setMessage] = useState<{
    type: 'success' | 'error';
    text: string;
  } | null>(null);
  const [showUseTemplateModal, setShowUseTemplateModal] = useState<Template | null>(null);

  // Filter templates based on search query
  const filteredTemplates = useMemo(() => {
    if (!templates) {
      return [];
    }
    if (!searchQuery.trim()) {
      return templates;
    }

    const query = searchQuery.toLowerCase();
    return templates.filter(
      (template) =>
        template.name.toLowerCase().includes(query) ||
        template.description?.toLowerCase().includes(query) ||
        template.type.toLowerCase().includes(query) ||
        template.tags?.some((tag) => tag.toLowerCase().includes(query)),
    );
  }, [templates, searchQuery]);

  // Group templates by type for display
  const templatesByType = useMemo(() => {
    const grouped: Record<string, Template[]> = {};
    for (const template of filteredTemplates) {
      const type = template.type || 'custom';
      if (!grouped[type]) {
        grouped[type] = [];
      }
      grouped[type].push(template);
    }
    return grouped;
  }, [filteredTemplates]);

  // Handle viewing a template
  const handleViewTemplate = useCallback(async (template: Template) => {
    setSelectedTemplate(template);
    setTemplateContent(null);
    setContentLoading(true);
    setContentError(null);

    try {
      const content = await fetchTemplateContent(template.name);
      setTemplateContent(content);
    } catch (err) {
      setContentError(err as Error);
    } finally {
      setContentLoading(false);
    }
  }, []);

  // Handle opening the use template modal
  const handleUseTemplate = useCallback((template: Template) => {
    setShowUseTemplateModal(template);
  }, []);

  // Handle confirming the use template action
  const handleConfirmUseTemplate = useCallback(
    async (configName: string) => {
      if (!showUseTemplateModal) {
        return;
      }

      const template = showUseTemplateModal;
      setShowUseTemplateModal(null);
      setMessage(null);

      try {
        const response = await applyTemplate({
          templateName: template.name,
          newConfigName: configName || undefined,
        });

        setMessage({
          type: 'success',
          text: response.message || `Configuration created at ${response.configPath}`,
        });

        // Close the preview modal if open
        setSelectedTemplate(null);
      } catch (err) {
        setMessage({
          type: 'error',
          text: getErrorMessage(err) || 'Failed to create configuration from template',
        });
      }
    },
    [showUseTemplateModal],
  );

  // Handle closing the preview modal
  const handleClosePreview = useCallback(() => {
    setSelectedTemplate(null);
    setTemplateContent(null);
    setContentError(null);
  }, []);

  // Handle copying template content
  const handleCopyContent = useCallback(async () => {
    if (!templateContent) {
      return;
    }

    try {
      await navigator.clipboard.writeText(templateContent.content);
      setMessage({
        type: 'success',
        text: 'Template content copied to clipboard',
      });
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement('textarea');
      textarea.value = templateContent.content;
      document.body.appendChild(textarea);
      textarea.select();
      const copied = document.execCommand('copy');
      document.body.removeChild(textarea);
      setMessage({
        type: copied ? 'success' : 'error',
        text: copied ? 'Template content copied to clipboard' : 'Failed to copy to clipboard',
      });
    }
  }, [templateContent]);

  return (
    <div className="space-y-6">
      {/* Header section with search and upload */}
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
            <Button
              tone="violet"
              leftIcon={<Upload className="h-4 w-4" />}
              onClick={() => setShowUploadModal(true)}
            >
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
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 py-3 pl-10 pr-10 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => setSearchQuery('')}
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
                onClick={() => setMessage(null)}
                className="ml-auto text-current hover:opacity-70"
                aria-label="Dismiss message"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Loading state */}
      {templatesLoading && !templates && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="flex items-center justify-center py-12">
            <div className="flex items-center gap-3 text-gray-400">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-violet-500 border-t-transparent" />
              <span>Loading templates...</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Error state */}
      {templatesError && (
        <Card className="border-red-500/30 bg-red-900/20">
          <CardContent className="space-y-3">
            <div className="flex items-start gap-3">
              <AlertCircle className="mt-1 h-5 w-5 text-red-400" />
              <div>
                <p className="font-semibold text-red-200">Failed to Load Templates</p>
                <SmallText className="text-red-300/90">{templatesError.message}</SmallText>
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-3"
                  onClick={() => refetchTemplates()}
                >
                  Retry
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Empty state */}
      {templates && templates.length === 0 && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="py-12 text-center">
            <FileCode className="mx-auto h-12 w-12 text-gray-600" />
            <H2 className="mt-4 mb-2">No Templates Available</H2>
            <P className="text-gray-400">
              Upload your first configuration template to get started.
            </P>
            <Button
              tone="violet"
              className="mt-4"
              leftIcon={<Upload className="h-4 w-4" />}
              onClick={() => setShowUploadModal(true)}
            >
              Upload Template
            </Button>
          </CardContent>
        </Card>
      )}

      {/* No search results */}
      {templates && templates.length > 0 && filteredTemplates.length === 0 && (
        <Card className="border-white/5 bg-gray-900/70">
          <CardContent className="py-12 text-center">
            <Search className="mx-auto h-12 w-12 text-gray-600" />
            <H2 className="mt-4 mb-2">No Matching Templates</H2>
            <P className="text-gray-400">
              No templates match your search &quot;{searchQuery}&quot;. Try a different search term.
            </P>
            <Button variant="outline" className="mt-4" onClick={() => setSearchQuery('')}>
              Clear Search
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Template grid */}
      {filteredTemplates.length > 0 && (
        <div className="space-y-8">
          {Object.entries(templatesByType).map(([type, typeTemplates]) => (
            <div key={type} className="space-y-4">
              <div className="flex items-center gap-3">
                <H2 className="mb-0 capitalize">{type} Templates</H2>
                <Tag colorScheme="gray">{typeTemplates.length}</Tag>
              </div>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {typeTemplates.map((template) => (
                  <TemplateCard
                    key={template.name}
                    template={template}
                    onView={handleViewTemplate}
                    onUse={handleUseTemplate}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Template preview modal */}
      {selectedTemplate && (
        <TemplatePreviewModal
          template={selectedTemplate}
          content={templateContent}
          loading={contentLoading}
          error={contentError}
          onClose={handleClosePreview}
          onUse={handleUseTemplate}
          onCopy={handleCopyContent}
        />
      )}

      {/* Upload template modal */}
      {showUploadModal && (
        <UploadTemplateModal
          onClose={() => setShowUploadModal(false)}
          onSuccess={() => {
            setShowUploadModal(false);
            refetchTemplates();
            setMessage({
              type: 'success',
              text: 'Template uploaded successfully',
            });
          }}
          onError={(error) => {
            setMessage({ type: 'error', text: error.message });
          }}
        />
      )}

      {/* Use Template Modal */}
      <InputModal
        isOpen={!!showUseTemplateModal}
        onSubmit={handleConfirmUseTemplate}
        onCancel={() => setShowUseTemplateModal(null)}
        title="Create Configuration"
        message="Enter a name for the new configuration (or leave blank for default):"
        placeholder="e.g., my-config"
        defaultValue={showUseTemplateModal ? `${showUseTemplateModal.name}-config` : ''}
        submitLabel="Create"
        submitTone="violet"
      />
    </div>
  );
};
