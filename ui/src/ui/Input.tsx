import {
	type FC,
	forwardRef,
	type InputHTMLAttributes,
	type ReactNode,
	type TextareaHTMLAttributes,
} from "react";

// Base input styles
const inputBaseStyles =
	"w-full rounded-lg border bg-gray-950/60 text-white placeholder:text-gray-500 transition-all focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed";
const inputFocusStyles =
	"focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20";
const inputBorderStyles = "border-white/10 hover:border-white/20";

// Text Input
interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
	label?: string;
	error?: string;
	hint?: string;
	leftIcon?: ReactNode;
	rightIcon?: ReactNode;
	containerClassName?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
	(
		{
			label,
			error,
			hint,
			leftIcon,
			rightIcon,
			className = "",
			containerClassName = "",
			id,
			...props
		},
		ref,
	) => {
		const inputId = id || label?.toLowerCase().replace(/\s+/g, "-");
		const hasError = !!error;

		return (
			<div className={containerClassName}>
				{label && (
					<label
						htmlFor={inputId}
						className="block text-sm font-medium text-gray-300 mb-2"
					>
						{label}
					</label>
				)}
				<div className="relative">
					{leftIcon && (
						<div className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">
							{leftIcon}
						</div>
					)}
					<input
						ref={ref}
						id={inputId}
						className={`
              ${inputBaseStyles}
              ${hasError ? "border-red-500 focus:border-red-500 focus:ring-red-500/20" : `${inputBorderStyles} ${inputFocusStyles}`}
              ${leftIcon ? "pl-10" : "px-4"}
              ${rightIcon ? "pr-10" : "px-4"}
              py-2.5
              ${className}
            `}
						{...props}
					/>
					{rightIcon && (
						<div className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">
							{rightIcon}
						</div>
					)}
				</div>
				{(error || hint) && (
					<p
						className={`mt-1.5 text-sm ${hasError ? "text-red-400" : "text-gray-500"}`}
					>
						{error || hint}
					</p>
				)}
			</div>
		);
	},
);

Input.displayName = "Input";

// Textarea
interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
	label?: string;
	error?: string;
	hint?: string;
	containerClassName?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
	(
		{
			label,
			error,
			hint,
			className = "",
			containerClassName = "",
			id,
			...props
		},
		ref,
	) => {
		const textareaId = id || label?.toLowerCase().replace(/\s+/g, "-");
		const hasError = !!error;

		return (
			<div className={containerClassName}>
				{label && (
					<label
						htmlFor={textareaId}
						className="block text-sm font-medium text-gray-300 mb-2"
					>
						{label}
					</label>
				)}
				<textarea
					ref={ref}
					id={textareaId}
					className={`
            ${inputBaseStyles}
            ${hasError ? "border-red-500 focus:border-red-500 focus:ring-red-500/20" : `${inputBorderStyles} ${inputFocusStyles}`}
            px-4 py-2.5 min-h-[100px] resize-y
            ${className}
          `}
					{...props}
				/>
				{(error || hint) && (
					<p
						className={`mt-1.5 text-sm ${hasError ? "text-red-400" : "text-gray-500"}`}
					>
						{error || hint}
					</p>
				)}
			</div>
		);
	},
);

Textarea.displayName = "Textarea";

// Select
interface SelectOption {
	value: string;
	label: string;
	disabled?: boolean;
}

interface SelectProps
	extends Omit<InputHTMLAttributes<HTMLSelectElement>, "onChange"> {
	label?: string;
	error?: string;
	hint?: string;
	options: SelectOption[];
	placeholder?: string;
	containerClassName?: string;
	onChange?: (value: string) => void;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
	(
		{
			label,
			error,
			hint,
			options,
			placeholder,
			className = "",
			containerClassName = "",
			id,
			onChange,
			...props
		},
		ref,
	) => {
		const selectId = id || label?.toLowerCase().replace(/\s+/g, "-");
		const hasError = !!error;

		return (
			<div className={containerClassName}>
				{label && (
					<label
						htmlFor={selectId}
						className="block text-sm font-medium text-gray-300 mb-2"
					>
						{label}
					</label>
				)}
				<select
					ref={ref}
					id={selectId}
					className={`
            ${inputBaseStyles}
            ${hasError ? "border-red-500 focus:border-red-500 focus:ring-red-500/20" : `${inputBorderStyles} ${inputFocusStyles}`}
            px-4 py-2.5 appearance-none cursor-pointer
            bg-[url('data:image/svg+xml;charset=utf-8,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20fill%3D%22none%22%20viewBox%3D%220%200%2024%2024%22%20stroke%3D%22%239ca3af%22%3E%3Cpath%20stroke-linecap%3D%22round%22%20stroke-linejoin%3D%22round%22%20stroke-width%3D%222%22%20d%3D%22M19%209l-7%207-7-7%22%2F%3E%3C%2Fsvg%3E')]
            bg-[length:1.25rem] bg-[right_0.75rem_center] bg-no-repeat pr-10
            ${className}
          `}
					onChange={(e) => onChange?.(e.target.value)}
					{...props}
				>
					{placeholder && (
						<option value="" disabled={true}>
							{placeholder}
						</option>
					)}
					{options.map((option) => (
						<option
							key={option.value}
							value={option.value}
							disabled={option.disabled}
						>
							{option.label}
						</option>
					))}
				</select>
				{(error || hint) && (
					<p
						className={`mt-1.5 text-sm ${hasError ? "text-red-400" : "text-gray-500"}`}
					>
						{error || hint}
					</p>
				)}
			</div>
		);
	},
);

Select.displayName = "Select";

// Checkbox
interface CheckboxProps
	extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
	label: string;
	description?: string;
	containerClassName?: string;
}

export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
	(
		{
			label,
			description,
			className = "",
			containerClassName = "",
			id,
			...props
		},
		ref,
	) => {
		const checkboxId = id || label.toLowerCase().replace(/\s+/g, "-");

		return (
			<div className={`flex items-start gap-3 ${containerClassName}`}>
				<input
					ref={ref}
					type="checkbox"
					id={checkboxId}
					className={`
            mt-0.5 h-4 w-4 rounded border-gray-600 bg-gray-800 text-violet-500
            focus:ring-2 focus:ring-violet-500/50 focus:ring-offset-0
            transition-colors cursor-pointer
            ${className}
          `}
					{...props}
				/>
				<div>
					<label
						htmlFor={checkboxId}
						className="text-sm font-medium text-gray-200 cursor-pointer"
					>
						{label}
					</label>
					{description && (
						<p className="text-sm text-gray-500 mt-0.5">{description}</p>
					)}
				</div>
			</div>
		);
	},
);

Checkbox.displayName = "Checkbox";

// Toggle/Switch
interface ToggleProps
	extends Omit<InputHTMLAttributes<HTMLInputElement>, "type"> {
	label: string;
	description?: string;
	containerClassName?: string;
}

export const Toggle = forwardRef<HTMLInputElement, ToggleProps>(
	(
		{
			label,
			description,
			className = "",
			containerClassName = "",
			id,
			checked,
			...props
		},
		ref,
	) => {
		const toggleId = id || label.toLowerCase().replace(/\s+/g, "-");

		return (
			<div
				className={`flex items-center justify-between gap-4 ${containerClassName}`}
			>
				<div>
					<label
						htmlFor={toggleId}
						className="text-sm font-medium text-gray-200 cursor-pointer"
					>
						{label}
					</label>
					{description && (
						<p className="text-sm text-gray-500 mt-0.5">{description}</p>
					)}
				</div>
				<button
					type="button"
					role="switch"
					aria-checked={checked}
					onClick={() => {
						const input = document.getElementById(toggleId) as HTMLInputElement;
						if (input) {
							input.click();
						}
					}}
					className={`
            relative inline-flex h-6 w-11 items-center rounded-full transition-colors
            focus:outline-none focus:ring-2 focus:ring-violet-500/50 focus:ring-offset-2 focus:ring-offset-gray-900
            ${checked ? "bg-violet-600" : "bg-gray-700"}
            ${className}
          `}
				>
					<span
						className={`
              inline-block h-4 w-4 transform rounded-full bg-white shadow-lg transition-transform
              ${checked ? "translate-x-6" : "translate-x-1"}
            `}
					/>
				</button>
				<input
					ref={ref}
					type="checkbox"
					id={toggleId}
					checked={checked}
					className="sr-only"
					{...props}
				/>
			</div>
		);
	},
);

Toggle.displayName = "Toggle";

// Form group wrapper
interface FormGroupProps {
	children: ReactNode;
	className?: string;
}

export const FormGroup: FC<FormGroupProps> = ({ children, className = "" }) => (
	<div className={`space-y-4 ${className}`}>{children}</div>
);

// Form section with title
interface FormSectionProps {
	title: string;
	description?: string;
	children: ReactNode;
	className?: string;
}

export const FormSection: FC<FormSectionProps> = ({
	title,
	description,
	children,
	className = "",
}) => (
	<div className={`space-y-4 ${className}`}>
		<div>
			<h3 className="text-lg font-semibold text-white">{title}</h3>
			{description && (
				<p className="text-sm text-gray-400 mt-1">{description}</p>
			)}
		</div>
		<div className="space-y-4">{children}</div>
	</div>
);
