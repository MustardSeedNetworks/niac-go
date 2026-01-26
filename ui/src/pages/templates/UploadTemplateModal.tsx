import { Upload, X } from 'lucide-react';
import { type FC, useState } from 'react';
import { uploadTemplate } from '../../api/client';
import type { Template } from '../../api/types';
import { Button } from '../../ui/Button';
import { SmallText } from '../../ui/Typography';

interface UploadTemplateModalProps {
  onClose: () => void;
  onSuccess: () => void;
  onError: (error: Error) => void;
}

export const UploadTemplateModal: FC<UploadTemplateModalProps> = ({
  onClose,
  onSuccess,
  onError,
}) => {
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
    const MaxSize = 1024 * 1024;
    if (file.size > MaxSize) {
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
    if (!(name.trim() && content.trim())) {
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
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <button
        type="button"
        className="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={onClose}
        aria-label="Close upload modal"
      />
      <div
        className="relative mx-4 max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-labelledby="upload-modal-title"
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
              <SmallText className="text-gray-400">Add a new configuration template</SmallText>
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

        {/* Modal Content */}
        <div className="space-y-4 p-6">
          {/* File upload */}
          <div>
            <label
              htmlFor="template-upload"
              className="mb-2 block text-sm font-medium text-gray-300"
            >
              Upload YAML File
            </label>
            <input
              id="template-upload"
              type="file"
              accept=".yaml,.yml"
              onChange={(e) => handleFileChange(e.target.files?.[0] || null)}
              className="w-full cursor-pointer rounded-lg border border-dashed border-white/20 bg-gray-950/40 p-3 text-sm text-white file:mr-3 file:rounded-md file:border-0 file:bg-violet-600 file:px-3 file:py-1 file:text-sm file:font-medium"
            />
            {uploadFile && (
              <SmallText className="mt-1 text-green-400">Selected: {uploadFile.name}</SmallText>
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
            <label
              htmlFor="template-description"
              className="mb-2 block text-sm font-medium text-gray-300"
            >
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
            <label
              htmlFor="template-content"
              className="mb-2 block text-sm font-medium text-gray-300"
            >
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
