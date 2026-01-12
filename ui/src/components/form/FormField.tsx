import { HelpCircle } from "lucide-react";
import type { FC, ReactNode } from "react";

export interface FormFieldProps {
  label: string;
  children: ReactNode;
  helpText?: string;
  required?: boolean;
  className?: string;
}

export const FormField: FC<FormFieldProps> = ({
  label,
  children,
  helpText,
  required,
  className = "",
}) => {
  return (
    <div className={className}>
      <label className="flex items-center gap-2 text-sm font-medium text-gray-300 mb-2">
        {label}
        {required && <span className="text-red-400">*</span>}
        {helpText && (
          <span className="relative group">
            <HelpCircle className="h-3.5 w-3.5 text-gray-500 cursor-help" />
            <span className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-3 py-1.5 bg-gray-800 text-gray-200 text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10 pointer-events-none">
              {helpText}
            </span>
          </span>
        )}
      </label>
      {children}
    </div>
  );
};
