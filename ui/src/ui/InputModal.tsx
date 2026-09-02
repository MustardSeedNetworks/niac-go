import { type FC, useEffect, useId, useRef, useState } from 'react';
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

  // The heading sits above a message, a layout Modal's own header cannot
  // express, so its id is what names the dialog. The input is labelled from the
  // same heading — it had no label at all, which axe reports as `label` and a
  // screen reader reads as an unnamed text field (#1668).
  const titleId = useId();
  const messageId = useId();

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
    <Modal
      isOpen={isOpen}
      onClose={onCancel}
      size="sm"
      showCloseButton={false}
      labelledBy={titleId}
    >
      <div className="stack-lg">
        <div>
          <h2 id={titleId} className="heading-3 text-text-primary">
            {title}
          </h2>
          <p id={messageId} className="text-text-secondary mt-tight">
            {message}
          </p>
        </div>
        <input
          ref={inputRef}
          type="text"
          aria-labelledby={titleId}
          aria-describedby={messageId}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          className="w-full rounded-lg border border-surface-border bg-bg-base/60 pad-sm text-sm text-text-primary placeholder:text-text-muted focus:border-brand-accent focus:outline-none"
        />
        <div className="flex justify-end gap-default pt-2">
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
