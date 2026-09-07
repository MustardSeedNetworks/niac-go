import { Copy, Download, FileCode, Pencil } from 'lucide-react';
import { type FC, useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router';
import { applyTemplate } from '../api/client';
import type { Template, TemplateContent } from '../api/types';
import { iconSizes } from '../constants/sizes';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
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
  const { t } = useTranslation('pages');
  const navigate = useNavigate();
  const titleId = useId();
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
      setEditError(getErrorMessage(err) || t('templates.previewModal.editCopyFailed'));
    } finally {
      setEditing(false);
    }
  };

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
    <Modal
      isOpen
      onClose={onClose}
      size="full"
      labelledBy={titleId}
      header={
        <div className="flex items-center gap-default">
          <div className="rounded-lg bg-brand-primary/20 pad-xs">
            <FileCode className={`${iconSizes.lg} text-brand-accent`} />
          </div>
          <div>
            <h2 id={titleId} className="heading-3 text-text-primary">
              {template.name}
            </h2>
            <SmallText className="text-text-muted">
              {template.description || t('templates.previewModal.defaultDescription')}
            </SmallText>
          </div>
        </div>
      }
      footer={
        <div className="flex w-full flex-wrap items-center justify-between gap-default">
          <div className="flex gap-compact">
            <Button
              variant="ghost"
              size="sm"
              leftIcon={<Copy className={iconSizes.md} />}
              onClick={onCopy}
              disabled={!content}
            >
              {t('templates.previewModal.copyYaml')}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              leftIcon={<Download className={iconSizes.md} />}
              onClick={handleDownload}
              disabled={!content}
            >
              {t('templates.previewModal.download')}
            </Button>
          </div>
          <div className="flex flex-col items-end gap-tight">
            <div className="flex gap-compact">
              <Button variant="outline" onClick={onClose}>
                {t('templates.previewModal.closeButton')}
              </Button>
              <Button
                variant="outline"
                leftIcon={<Pencil className={iconSizes.md} />}
                onClick={handleEditCopy}
                disabled={editing || !content}
                title={t('templates.previewModal.editCopyTitle')}
              >
                {editing
                  ? t('templates.previewModal.cloning')
                  : t('templates.previewModal.editCopy')}
              </Button>
              <Button
                tone="violet"
                onClick={() => onUse(template)}
                title={t('templates.previewModal.pickTemplateTitle')}
              >
                {t('templates.previewModal.pickTemplate')}
              </Button>
            </div>
            {editError && (
              <SmallText className="text-status-error" role="alert">
                {editError}
              </SmallText>
            )}
          </div>
        </div>
      }
    >
      <div className="stack-lg">
        <div className="flex flex-wrap items-center gap-default">
          <Tag colorScheme="purple">
            {t('templates.previewModal.deviceCount', { count: template.deviceCount })}
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

        {loading && (
          <div className="flex-center py-centered">
            <div className="flex items-center gap-default text-text-muted">
              <div className="h-5 w-5 animate-spin rounded-full border-2 border-brand-primary border-t-transparent" />
              <span>{t('templates.previewModal.loading')}</span>
            </div>
          </div>
        )}

        {error && (
          <div className="rounded-lg border border-status-error/30 bg-status-error/10 pad text-status-error">
            <p className="font-semibold">{t('templates.previewModal.loadFailed')}</p>
            <SmallText className="text-status-error">{error.message}</SmallText>
          </div>
        )}

        {content && !loading && !error && (
          <YamlViewer
            ariaLabel={t('templates.previewModal.yamlEditorAria')}
            value={content.content}
            height="auto"
            minHeight="200px"
            maxHeight="400px"
            showLineNumbers={true}
            showFoldGutter={true}
          />
        )}
      </div>
    </Modal>
  );
};
