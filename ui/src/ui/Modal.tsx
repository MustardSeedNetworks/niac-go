import { X } from "lucide-react";
import {
	type FC,
	type KeyboardEvent,
	type ReactNode,
	useCallback,
	useEffect,
} from "react";

export type ModalSize = "sm" | "md" | "lg" | "xl" | "full";

export interface ModalProps {
	isOpen: boolean;
	onClose: () => void;
	title?: string;
	children: ReactNode;
	size?: ModalSize;
	showCloseButton?: boolean;
	closeOnBackdropClick?: boolean;
	closeOnEscape?: boolean;
	className?: string;
}

const sizeClasses: Record<ModalSize, string> = {
	sm: "max-w-sm",
	md: "max-w-md",
	lg: "max-w-lg",
	xl: "max-w-xl",
	full: "max-w-4xl",
};

export const Modal: FC<ModalProps> = ({
	isOpen,
	onClose,
	title,
	children,
	size = "md",
	showCloseButton = true,
	closeOnBackdropClick = true,
	closeOnEscape = true,
	className = "",
}) => {
	// Handle escape key
	const handleKeyDown = useCallback(
		(e: globalThis.KeyboardEvent) => {
			if (closeOnEscape && e.key === "Escape") {
				onClose();
			}
		},
		[closeOnEscape, onClose],
	);

	useEffect(() => {
		if (isOpen) {
			document.addEventListener("keydown", handleKeyDown);
			// Prevent body scroll when modal is open
			document.body.style.overflow = "hidden";
		}
		return () => {
			document.removeEventListener("keydown", handleKeyDown);
			document.body.style.overflow = "";
		};
	}, [isOpen, handleKeyDown]);

	if (!isOpen) {
		return null;
	}

	const handleContentKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
		// Prevent escape from bubbling if handled
		if (e.key === "Escape") {
			e.stopPropagation();
		}
	};

	return (
		<div className="fixed inset-0 z-50 flex items-center justify-center">
			{closeOnBackdropClick ? (
				<button
					type="button"
					className="absolute inset-0 bg-black/70 backdrop-blur-sm"
					onClick={onClose}
					aria-label="Close modal"
				/>
			) : (
				<div className="absolute inset-0 bg-black/70 backdrop-blur-sm" />
			)}
			<div
				className={`mx-4 w-full ${sizeClasses[size]} rounded-2xl border border-white/10 bg-gray-900/95 shadow-2xl ${className}`}
				role="dialog"
				aria-modal="true"
				aria-labelledby={title ? "modal-title" : undefined}
				onKeyDown={handleContentKeyDown}
			>
				{(title || showCloseButton) && (
					<div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
						{title && (
							<h2 id="modal-title" className="text-lg font-semibold text-white">
								{title}
							</h2>
						)}
						{showCloseButton && (
							<button
								type="button"
								onClick={onClose}
								className="ml-auto p-1 text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/10"
								aria-label="Close modal"
							>
								<X className="h-5 w-5" />
							</button>
						)}
					</div>
				)}
				<div className="p-6">{children}</div>
			</div>
		</div>
	);
};

// Convenience components for modal sections
export const ModalHeader: FC<{ children: ReactNode; className?: string }> = ({
	children,
	className = "",
}) => <div className={`mb-4 ${className}`}>{children}</div>;

export const ModalBody: FC<{ children: ReactNode; className?: string }> = ({
	children,
	className = "",
}) => <div className={`space-y-4 ${className}`}>{children}</div>;

export const ModalFooter: FC<{ children: ReactNode; className?: string }> = ({
	children,
	className = "",
}) => (
	<div
		className={`flex justify-end gap-3 pt-4 mt-4 border-t border-white/10 ${className}`}
	>
		{children}
	</div>
);
