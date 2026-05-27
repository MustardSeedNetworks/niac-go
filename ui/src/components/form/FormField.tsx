import { HelpCircle } from 'lucide-react';
import type { FC, ReactNode } from 'react';
import { iconSizes } from '../../constants/sizes';

export interface FormFieldProps {
  label: string;
  children: ReactNode;
  helpText?: string;
  required?: boolean;
  className?: string;
  htmlFor?: string;
}

export const FormField: FC<FormFieldProps> = ({
  label,
  children,
  helpText,
  required,
  className = '',
  htmlFor,
}) => (
  <div className={className}>
    {htmlFor ? (
      <label
        htmlFor={htmlFor}
        className="flex items-center gap-compact text-sm font-medium text-text-secondary mb-2"
      >
        {label}
        {required && <span className="text-status-error">*</span>}
        {helpText && (
          <span className="relative group">
            <HelpCircle className={`${iconSizes.sm} text-text-muted cursor-help`} />
            <span className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-compact-md bg-bg-elevated text-text-primary text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 pointer-events-none">
              {helpText}
            </span>
          </span>
        )}
      </label>
    ) : (
      <div className="flex items-center gap-compact text-sm font-medium text-text-secondary mb-2">
        {label}
        {required && <span className="text-status-error">*</span>}
        {helpText && (
          <span className="relative group">
            <HelpCircle className={`${iconSizes.sm} text-text-muted cursor-help`} />
            <span className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-compact-md bg-bg-elevated text-text-primary text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 pointer-events-none">
              {helpText}
            </span>
          </span>
        )}
      </div>
    )}
    {children}
  </div>
);
