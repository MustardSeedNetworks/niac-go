import { useCallback, useMemo, useState } from 'react';
import { applyTemplate, fetchTemplateContent, fetchTemplates } from '../../api/client';
import type { Template, TemplateContent } from '../../api/types';
import { useApiResource } from '../../hooks/useApiResource';
import { getErrorMessage } from '../../utils/format';

export interface StatusMessage {
  type: 'success' | 'error';
  text: string;
}

export interface UseTemplatesReturn {
  // Data
  templates: Template[] | null;
  filteredTemplates: Template[];
  templatesByType: Record<string, Template[]>;
  // Loading/Error states
  templatesLoading: boolean;
  templatesError: Error | null;
  // Search
  searchQuery: string;
  setSearchQuery: (query: string) => void;
  // Selected template
  selectedTemplate: Template | null;
  templateContent: TemplateContent | null;
  contentLoading: boolean;
  contentError: Error | null;
  // Modals
  showUploadModal: boolean;
  setShowUploadModal: (show: boolean) => void;
  showUseTemplateModal: Template | null;
  setShowUseTemplateModal: (template: Template | null) => void;
  // Messages
  message: StatusMessage | null;
  setMessage: (msg: StatusMessage | null) => void;
  // Actions
  handleViewTemplate: (template: Template) => Promise<void>;
  handleUseTemplate: (template: Template) => void;
  handleConfirmUseTemplate: (configName: string) => Promise<void>;
  handleClosePreview: () => void;
  handleCopyContent: () => Promise<void>;
  refetchTemplates: () => void;
}

export function useTemplates(): UseTemplatesReturn {
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
  const [message, setMessage] = useState<StatusMessage | null>(null);
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
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setMessage({
        type: 'success',
        text: 'Template content copied to clipboard',
      });
    }
  }, [templateContent]);

  return {
    templates,
    filteredTemplates,
    templatesByType,
    templatesLoading,
    templatesError,
    searchQuery,
    setSearchQuery,
    selectedTemplate,
    templateContent,
    contentLoading,
    contentError,
    showUploadModal,
    setShowUploadModal,
    showUseTemplateModal,
    setShowUseTemplateModal,
    message,
    setMessage,
    handleViewTemplate,
    handleUseTemplate,
    handleConfirmUseTemplate,
    handleClosePreview,
    handleCopyContent,
    refetchTemplates,
  };
}
