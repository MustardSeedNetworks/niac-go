import { type FC, useEffect, useRef, useState } from 'react';
import { Button } from './Button';
import { Modal } from './Modal';

export interface InputModalProps {
  isOpen: boolean;
  onSubmit: (value: string) => void;
  onCancel: () => void;
  title: string;
  message: string;
  placeholder?: string;
  defaultValue?: string;
  submitLabel?: string;
  cancelLabel?: string;
  submitTone?: 'violet' | 'blue' | 'green' | 'red';
}

export const InputModal: FC<InputModalProps> = ({
  isOpen,
  onSubmit,
  onCancel,
  title,
  message,
  placeholder = '',
  defaultValue = '',
  submitLabel = 'Submit',
  cancelLabel = 'Cancel',
  submitTone = 'violet',
}) => {
  const [value, setValue] = useState(defaultValue);
  const inputRef = useRef<HTMLInputElement>(null);

  // Reset value and focus input when modal opens
  useEffect(() => {
    if (isOpen) {
      setValue(defaultValue);
      // Focus input after modal animation
      setTimeout(() => {
        inputRef.current?.focus();
        inputRef.current?.select();
      }, 100);
    }
  }, [isOpen, defaultValue]);

  const handleSubmit = () => {
    onSubmit(value);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onCancel} size="sm" showCloseButton={false}>
      <div className="space-y-4">
        <div>
          <h2 className="text-lg font-semibold text-white">{title}</h2>
          <p className="text-gray-300 mt-1">{message}</p>
        </div>
        <input
          ref={inputRef}
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none"
        />
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="outline" onClick={onCancel}>
            {cancelLabel}
          </Button>
          <Button tone={submitTone} onClick={handleSubmit}>
            {submitLabel}
          </Button>
        </div>
      </div>
    </Modal>
  );
};
