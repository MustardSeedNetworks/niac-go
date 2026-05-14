import { Copy, Download, FileCode, Pencil, X } from 'lucide-react';
import { type FC, useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { applyTemplate } from '../api/client';
import type { Template, TemplateContent } from '../api/types';
import { Button } from '../ui/Button';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { getErrorMessage } from '../utils/format';
import { YamlViewer } from './config/YamlEditor';

interface TemplatePreviewModalProps {
  template: Template | null;
  content: TemplateContent | null;
  loading: boolean;
  error: Error | null;
  onClose: () => void;
  onUse: (template: Template) => void;
  onCopy: () => void;
}

export const TemplatePreviewModal: FC<TemplatePreviewModalProps> = ({
  template,
  content,
  loading,
  error,
  onClose,
  onUse,
  onCopy,
}) => {
  const navigate = useNavigate();
  const [editing, setEditing] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  // Edit a copy: clone the template to a fresh user config and jump the
  // user into the Device Library so they can tweak it before running. The
  // first device in the cloned config is opened in the visual editor so
  // there's something concrete to land on.
  const handleEditCopy = async () => {
    if (!template || editing) return;
    setEditing(true);
    setEditError(null);
    try {
      const newName = `${template.name}-edit`;
      await applyTemplate({
        templateName: template.name,
        newConfigName: newName,
      });
      onClose();
      // Land on the Device Library. The user picks a device to enter
      // the editor; the new config is now in the saved-configs list.
      navigate('/device-config');
    } catch (err) {
      setEditError(getErrorMessage(err) || 'Failed to create editable copy');
    } finally {
      setEditing(false);
    }
  };

  // Handle escape key to close modal
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    },
    [onClose],
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    // Prevent body scroll when modal is open
    document.body.style.overflow = 'hidden';

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [handleKeyDown]);

  if (!template) {
    return null;
  }

  const handleDownload = () => {
    if (!content) {
      return;
    }

    const blob = new Blob([content.content], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${template.name}.yaml`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onClose}
        aria-label="Close modal"
      />
      <div
        className="relative mx-4 max-h-[90vh] w-full max-w-4xl overflow-hidden rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
      >
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-white/10 px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="rounded-lg bg-violet-500/20 p-2">
              <FileCode className="h-5 w-5 text-violet-300" />
            </div>
            <div>
              <h2 id="modal-title" className="text-lg font-semibold text-white">
                {template.name}
              </h2>
              <SmallText className="text-gray-400">
                {template.description || 'Configuration template preview'}
              </SmallText>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-gray-400 hover:bg-white/10 hover:text-white transition-colors"
            aria-label="Close modal"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Template metadata */}
        <div className="flex flex-wrap items-center gap-3 border-b border-white/10 px-6 py-3 bg-gray-950/50">
          <Tag colorScheme="purple">
            {template.deviceCount} {template.deviceCount === 1 ? 'device' : 'devices'}
          </Tag>
          <Tag colorScheme="gray" className="capitalize">
            {template.type}
          </Tag>
          {content && (
            <Tag colorScheme="blue" className="uppercase">
              {content.format}
            </Tag>
          )}
          {template.tags?.map((tag) => (
            <Tag key={tag} colorScheme="gray" className="text-xs">
              {tag}
            </Tag>
          ))}
        </div>

        {/* Content area */}
        <div className="max-h-[50vh] overflow-auto p-6">
          {loading && (
            <div className="flex items-center justify-center py-12">
              <div className="flex items-center gap-3 text-gray-400">
                <div className="h-5 w-5 animate-spin rounded-full border-2 border-violet-500 border-t-transparent" />
                <span>Loading template content...</span>
              </div>
            </div>
          )}

          {error && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4 text-red-300">
              <p className="font-semibold">Failed to load template</p>
              <SmallText className="text-red-400">{error.message}</SmallText>
            </div>
          )}

          {content && !loading && !error && (
            <YamlViewer
              value={content.content}
              height="auto"
              minHeight="200px"
              maxHeight="400px"
              showLineNumbers={true}
              showFoldGutter={true}
            />
          )}
        </div>

        {/* Modal Footer */}
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-white/10 px-6 py-4 bg-gray-950/50">
          <div className="flex gap-2">
            <Button
              variant="ghost"
              size="sm"
              leftIcon={<Copy className="h-4 w-4" />}
              onClick={onCopy}
              disabled={!content}
            >
              Copy YAML
            </Button>
            <Button
              variant="ghost"
              size="sm"
              leftIcon={<Download className="h-4 w-4" />}
              onClick={handleDownload}
              disabled={!content}
            >
              Download
            </Button>
          </div>
          <div className="flex flex-col items-end gap-1">
            <div className="flex gap-2">
              <Button variant="outline" onClick={onClose}>
                Close
              </Button>
              <Button
                variant="outline"
                leftIcon={<Pencil className="h-4 w-4" />}
                onClick={handleEditCopy}
                disabled={editing || !content}
                title="Clone this template to your saved configs and open the device editor"
              >
                {editing ? 'Cloning…' : 'Edit a copy'}
              </Button>
              <Button tone="violet" onClick={() => onUse(template)}>
                Pick this template
              </Button>
            </div>
            {editError && (
              <SmallText className="text-red-300" role="alert">
                {editError}
              </SmallText>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};
