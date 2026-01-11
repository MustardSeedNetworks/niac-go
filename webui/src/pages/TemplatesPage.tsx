import { type FC, useState, useMemo, useCallback } from 'react';
import { Search, Upload, X, FileCode, AlertCircle } from 'lucide-react';
import {
  Card,
  CardContent,
  Button,
  H2,
  P,
  SmallText,
  Tag,
} from '../ui';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchTemplates,
  fetchTemplateContent,
  applyTemplate,
  uploadTemplate,
} from '../api/client';
import type { Template, TemplateContent } from '../api/types';
import { TemplateCard } from '../components/TemplateCard';
import { TemplatePreviewModal } from '../components/TemplatePreviewModal';

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
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  // Filter templates based on search query
  const filteredTemplates = useMemo(() => {
    if (!templates) return [];
    if (!searchQuery.trim()) return templates;

    const query = searchQuery.toLowerCase();
    return templates.filter(
      (template) =>
        template.name.toLowerCase().includes(query) ||
        template.description?.toLowerCase().includes(query) ||
        template.type.toLowerCase().includes(query) ||
        template.tags?.some((tag) => tag.toLowerCase().includes(query))
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

  // Handle using a template
  const handleUseTemplate = useCallback(async (template: Template) => {
    const configName = prompt(
      `Enter a name for the new configuration (or leave blank for default):`,
      `${template.name}-config`
    );

    if (configName === null) return; // User cancelled

    setMessage(null);

    try {
      const response = await applyTemplate({
        template_name: template.name,
        new_config_name: configName || undefined,
      });

      setMessage({
        type: 'success',
        text: response.message || `Configuration created at ${response.config_path}`,
      });

      // Close the preview modal if open
      setSelectedTemplate(null);
    } catch (err) {
      setMessage({
        type: 'error',
        text: (err as Error).message || 'Failed to create configuration from template',
      });
    }
  }, []);

  // Handle closing the preview modal
  const handleClosePreview = useCallback(() => {
    setSelectedTemplate(null);
    setTemplateContent(null);
    setContentError(null);
  }, []);

  // Handle copying template content
  const handleCopyContent = useCallback(async () => {
    if (!templateContent) return;

    try {
      await navigator.clipboard.writeText(templateContent.content);
      setMessage({ type: 'success', text: 'Template content copied to clipboard' });
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement('textarea');
      textarea.value = templateContent.content;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setMessage({ type: 'success', text: 'Template content copied to clipboard' });
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
            setMessage({ type: 'success', text: 'Template uploaded successfully' });
          }}
          onError={(error) => {
            setMessage({ type: 'error', text: error.message });
          }}
        />
      )}
    </div>
  );
};

// Upload Template Modal Component
interface UploadTemplateModalProps {
  onClose: () => void;
  onSuccess: () => void;
  onError: (error: Error) => void;
}

const UploadTemplateModal: FC<UploadTemplateModalProps> = ({ onClose, onSuccess, onError }) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [content, setContent] = useState('');
  const [type, setType] = useState<Template['type']>('custom');
  const [uploading, setUploading] = useState(false);
  const [uploadFile, setUploadFile] = useState<File | null>(null);

  const handleFileChange = async (file: File | null) => {
    if (!file) {
      setUploadFile(null);
      return;
    }

    // Validate file size (1MB limit for templates)
    const MAX_SIZE = 1024 * 1024;
    if (file.size > MAX_SIZE) {
      onError(new Error('File too large. Maximum size is 1MB'));
      return;
    }

    // Validate file type
    if (!file.name.match(/\.(yaml|yml)$/i)) {
      onError(new Error('Please select a YAML file (.yaml or .yml)'));
      return;
    }

    setUploadFile(file);

    // Read file content
    try {
      const text = await file.text();
      setContent(text);

      // Auto-fill name from filename if empty
      if (!name) {
        setName(file.name.replace(/\.(yaml|yml)$/i, ''));
      }
    } catch {
      onError(new Error('Failed to read file'));
    }
  };

  const handleSubmit = async () => {
    if (!name.trim() || !content.trim()) {
      onError(new Error('Name and content are required'));
      return;
    }

    setUploading(true);

    try {
      await uploadTemplate({
        name: name.trim(),
        description: description.trim(),
        content,
        type,
      });
      onSuccess();
    } catch (err) {
      onError(err as Error);
    } finally {
      setUploading(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="upload-modal-title"
    >
      <div
        className="relative mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-white/10 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-violet-500/20 p-2">
              <Upload className="h-5 w-5 text-violet-300" />
            </div>
            <div>
              <h2 id="upload-modal-title" className="text-lg font-semibold text-white">
                Upload Template
              </h2>
              <SmallText className="text-gray-400">
                Add a new configuration template
              </SmallText>
            </div>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-2 text-gray-400 hover:bg-white/10 hover:text-white transition-colors"
            aria-label="Close modal"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Modal Content */}
        <div className="space-y-4 p-6">
          {/* File upload */}
          <div>
            <label className="mb-2 block text-sm font-medium text-gray-300">
              Upload YAML File
            </label>
            <input
              type="file"
              accept=".yaml,.yml"
              onChange={(e) => handleFileChange(e.target.files?.[0] || null)}
              className="w-full cursor-pointer rounded-lg border border-dashed border-white/20 bg-gray-950/40 p-3 text-sm text-white file:mr-3 file:rounded-md file:border-0 file:bg-violet-600 file:px-3 file:py-1 file:text-sm file:font-medium"
            />
            {uploadFile && (
              <SmallText className="mt-1 text-green-400">
                Selected: {uploadFile.name}
              </SmallText>
            )}
          </div>

          {/* Name input */}
          <div>
            <label htmlFor="template-name" className="mb-2 block text-sm font-medium text-gray-300">
              Template Name *
            </label>
            <input
              id="template-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Basic Network"
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
            />
          </div>

          {/* Description input */}
          <div>
            <label htmlFor="template-description" className="mb-2 block text-sm font-medium text-gray-300">
              Description
            </label>
            <textarea
              id="template-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of this template..."
              rows={2}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none resize-none"
            />
          </div>

          {/* Type selector */}
          <div>
            <label htmlFor="template-type" className="mb-2 block text-sm font-medium text-gray-300">
              Template Type
            </label>
            <select
              id="template-type"
              value={type}
              onChange={(e) => setType(e.target.value as Template['type'])}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white focus:border-violet-400 focus:outline-none"
            >
              <option value="basic">Basic Network</option>
              <option value="router">Router</option>
              <option value="switch">Switch</option>
              <option value="access-point">Access Point</option>
              <option value="server">Server</option>
              <option value="complete">Complete Network</option>
              <option value="custom">Custom</option>
            </select>
          </div>

          {/* Content textarea */}
          <div>
            <label htmlFor="template-content" className="mb-2 block text-sm font-medium text-gray-300">
              YAML Content *
            </label>
            <textarea
              id="template-content"
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="Paste your YAML configuration here..."
              rows={10}
              className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 font-mono text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none resize-none"
              spellCheck={false}
            />
          </div>
        </div>

        {/* Modal Footer */}
        <div className="flex justify-end gap-3 border-t border-white/10 px-6 py-4 bg-gray-950/50">
          <Button variant="outline" onClick={onClose} disabled={uploading}>
            Cancel
          </Button>
          <Button
            tone="violet"
            onClick={handleSubmit}
            disabled={uploading || !name.trim() || !content.trim()}
          >
            {uploading ? 'Uploading...' : 'Upload Template'}
          </Button>
        </div>
      </div>
    </div>
  );
};
