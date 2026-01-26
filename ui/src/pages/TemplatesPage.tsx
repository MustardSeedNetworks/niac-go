import type { FC } from 'react';
import { TemplatePreviewModal } from '../components/TemplatePreviewModal';
import { InputModal } from '../ui/InputModal';
import {
  TemplateGrid,
  TemplatesEmptyState,
  TemplatesErrorState,
  TemplatesHeader,
  TemplatesLoadingState,
  TemplatesNoResultsState,
  UploadTemplateModal,
  useTemplates,
} from './templates';

export const TemplatesPage: FC = () => {
  const {
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
  } = useTemplates();

  return (
    <div className="space-y-6">
      {/* Header section with search and upload */}
      <TemplatesHeader
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onUploadClick={() => setShowUploadModal(true)}
        message={message}
        onDismissMessage={() => setMessage(null)}
      />

      {/* Loading state */}
      <TemplatesLoadingState show={templatesLoading && !templates} />

      {/* Error state */}
      <TemplatesErrorState error={templatesError} onRetry={refetchTemplates} />

      {/* Empty state */}
      <TemplatesEmptyState
        show={!!templates && templates.length === 0}
        onUploadClick={() => setShowUploadModal(true)}
      />

      {/* No search results */}
      <TemplatesNoResultsState
        show={!!templates && templates.length > 0 && filteredTemplates.length === 0}
        searchQuery={searchQuery}
        onClearSearch={() => setSearchQuery('')}
      />

      {/* Template grid */}
      {filteredTemplates.length > 0 && (
        <TemplateGrid
          templatesByType={templatesByType}
          onView={handleViewTemplate}
          onUse={handleUseTemplate}
        />
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
