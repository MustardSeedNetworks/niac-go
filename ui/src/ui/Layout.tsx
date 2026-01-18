import type { LucideIcon } from "lucide-react";
import { ChevronDown, ChevronLeft, ChevronRight } from "lucide-react";
import {
	createElement,
	type FC,
	type ReactNode,
	useCallback,
	useEffect,
	useRef,
	useState,
} from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";

export interface NavItem {
	path: string;
	label: string;
	icon?: LucideIcon | ReactNode;
	badge?: string;
}

export interface NavGroup {
	label: string;
	items: NavItem[];
}

interface PageShellProps {
	children: ReactNode;
	className?: string;
}

export const PageShell: FC<PageShellProps> = ({ children, className = "" }) => (
	<div
		className={`min-h-screen bg-gradient-to-br from-gray-950 via-gray-900 to-gray-950 ${className}`}
	>
		<div className="mx-auto max-w-7xl px-4 sm:px-6 py-6">{children}</div>
	</div>
);

interface PrimaryNavProps {
	items: NavItem[];
	groups?: NavGroup[];
	currentPath?: string;
	onNavigate?: (path: string) => void;
	logo?: ReactNode;
	version?: string;
	className?: string;
}

export const PrimaryNav: FC<PrimaryNavProps> = ({
	items,
	groups,
	currentPath: externalPath,
	onNavigate: externalNavigate,
	logo,
	version,
	className = "",
}) => {
	const location = useLocation();
	const navigate = useNavigate();
	const scrollContainerRef = useRef<HTMLDivElement>(null);
	const [showLeftScroll, setShowLeftScroll] = useState(false);
	const [showRightScroll, setShowRightScroll] = useState(false);
	const [openGroup, setOpenGroup] = useState<string | null>(null);

	const currentPath = externalPath ?? location.pathname;
	const handleNavigate = externalNavigate ?? ((path: string) => navigate(path));

	// Check scroll position
	const checkScroll = useCallback(() => {
		const container = scrollContainerRef.current;
		if (container) {
			setShowLeftScroll(container.scrollLeft > 0);
			setShowRightScroll(
				container.scrollLeft <
					container.scrollWidth - container.clientWidth - 1,
			);
		}
	}, []);

	useEffect(() => {
		checkScroll();
		window.addEventListener("resize", checkScroll);
		return () => window.removeEventListener("resize", checkScroll);
	}, [checkScroll]);

	const scroll = (direction: "left" | "right") => {
		const container = scrollContainerRef.current;
		if (container) {
			const scrollAmount = 200;
			container.scrollBy({
				left: direction === "left" ? -scrollAmount : scrollAmount,
				behavior: "smooth",
			});
			setTimeout(checkScroll, 300);
		}
	};

	const renderNavItem = (item: NavItem, isGrouped = false) => {
		const isActive =
			currentPath === item.path ||
			(item.path !== "/" && currentPath.startsWith(item.path));

		const iconElement = item.icon
			? typeof item.icon === "function"
				? createElement(item.icon as LucideIcon, {
						className: "h-4 w-4 flex-shrink-0",
					})
				: item.icon
			: null;

		return (
			<button
				type="button"
				key={item.path}
				onClick={() => {
					handleNavigate(item.path);
					setOpenGroup(null);
				}}
				className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-all duration-200 whitespace-nowrap ${
					isActive
						? "bg-gradient-to-r from-violet-600/30 to-violet-500/20 text-white border-l-2 border-violet-500 shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]"
						: "text-gray-400 hover:text-white hover:bg-white/5"
				} ${isGrouped ? "w-full justify-start" : ""}`}
			>
				{iconElement}
				<span>{item.label}</span>
				{item.badge && (
					<span
						className={`ml-1 px-1.5 py-0.5 text-xs rounded font-medium ${
							item.badge === "New"
								? "bg-emerald-500/20 text-emerald-300"
								: item.badge === "Beta"
									? "bg-amber-500/20 text-amber-300"
									: "bg-violet-500/20 text-violet-300"
						}`}
					>
						{item.badge}
					</span>
				)}
			</button>
		);
	};

	// Use groups if provided, otherwise use flat items
	const navItems = groups ? groups.flatMap((g) => g.items) : items;

	return (
		<nav className={`relative flex items-center gap-2 mb-6 ${className}`}>
			{/* Logo */}
			{logo && <div className="flex-shrink-0 mr-4">{logo}</div>}

			{/* Scroll left button */}
			{showLeftScroll && (
				<button
					type="button"
					onClick={() => scroll("left")}
					className="flex-shrink-0 p-1.5 rounded-lg bg-gray-800/80 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors"
					aria-label="Scroll left"
				>
					<ChevronLeft className="h-4 w-4" />
				</button>
			)}

			{/* Navigation items */}
			<div
				ref={scrollContainerRef}
				className="flex items-center gap-1 overflow-x-auto scrollbar-hide scroll-smooth flex-1"
				onScroll={checkScroll}
			>
				{groups
					? // Grouped navigation
						groups.map((group, groupIndex) => (
							<div key={group.label} className="relative flex items-center">
								{groupIndex > 0 && (
									<div className="h-6 w-px bg-white/10 mx-2" />
								)}
								<div className="relative">
									<button
										type="button"
										onClick={() =>
											setOpenGroup(
												openGroup === group.label ? null : group.label,
											)
										}
										className="flex items-center gap-1 px-3 py-2 text-sm font-medium text-gray-400 hover:text-white transition-colors"
									>
										<span>{group.label}</span>
										<ChevronDown
											className={`h-3 w-3 transition-transform ${openGroup === group.label ? "rotate-180" : ""}`}
										/>
									</button>
									{openGroup === group.label && (
										<div className="absolute top-full left-0 mt-1 py-1 min-w-[180px] rounded-lg bg-gray-900/95 backdrop-blur-xl border border-white/10 shadow-xl z-50 animate-slide-down">
											{group.items.map((item) => (
												<div key={item.path} className="px-1">
													{renderNavItem(item, true)}
												</div>
											))}
										</div>
									)}
								</div>
							</div>
						))
					: // Flat navigation
						navItems.map((item) => renderNavItem(item))}
			</div>

			{/* Scroll right button */}
			{showRightScroll && (
				<button
					type="button"
					onClick={() => scroll("right")}
					className="flex-shrink-0 p-1.5 rounded-lg bg-gray-800/80 text-gray-400 hover:text-white hover:bg-gray-700 transition-colors"
					aria-label="Scroll right"
				>
					<ChevronRight className="h-4 w-4" />
				</button>
			)}

			{/* Version badge */}
			{version && (
				<div className="flex-shrink-0 ml-2 px-2 py-1 text-xs font-mono text-gray-500 bg-gray-800/50 rounded">
					{version}
				</div>
			)}
		</nav>
	);
};

interface PageHeaderProps {
	title: string;
	description?: string;
	icon?: LucideIcon;
	actions?: ReactNode;
	breadcrumbs?: { label: string; href?: string }[];
	className?: string;
}

export const PageHeader: FC<PageHeaderProps> = ({
	title,
	description,
	icon,
	actions,
	breadcrumbs,
	className = "",
}) => (
	<div className={`mb-6 animate-fade-in ${className}`}>
		{breadcrumbs && breadcrumbs.length > 0 && (
			<Breadcrumb items={breadcrumbs} className="mb-3" />
		)}
		<div className="flex flex-wrap items-start justify-between gap-4">
			<div className="flex items-center gap-3">
				{icon && createElement(icon, { className: "h-8 w-8 text-violet-400" })}
				<div>
					<h1 className="text-2xl font-bold text-white font-display">
						{title}
					</h1>
					{description && (
						<p className="text-sm text-gray-400 mt-1 max-w-2xl">
							{description}
						</p>
					)}
				</div>
			</div>
			{actions && <div className="flex items-center gap-3">{actions}</div>}
		</div>
	</div>
);

// Status indicator component
interface StatusIndicatorProps {
	status: "online" | "offline" | "warning" | "error" | "pending";
	label?: string;
	pulse?: boolean;
	size?: "sm" | "md" | "lg";
	className?: string;
}

const statusColors = {
	online: "bg-emerald-500",
	offline: "bg-gray-500",
	warning: "bg-amber-500",
	error: "bg-red-500",
	pending: "bg-blue-500",
};

const statusLabels = {
	online: "Online",
	offline: "Offline",
	warning: "Warning",
	error: "Error",
	pending: "Pending",
};

export const StatusIndicator: FC<StatusIndicatorProps> = ({
	status,
	label,
	pulse = false,
	size = "md",
	className = "",
}) => {
	const sizeClasses = {
		sm: "h-2 w-2",
		md: "h-2.5 w-2.5",
		lg: "h-3 w-3",
	};

	return (
		<div className={`flex items-center gap-2 ${className}`}>
			<span
				className={`relative inline-flex ${sizeClasses[size]} rounded-full ${statusColors[status]}`}
			>
				{pulse && (status === "online" || status === "pending") && (
					<span
						className={`absolute inset-0 rounded-full ${statusColors[status]} animate-ping opacity-75`}
					/>
				)}
			</span>
			{(label || label === undefined) && (
				<span className="text-sm text-gray-300">
					{label ?? statusLabels[status]}
				</span>
			)}
		</div>
	);
};

// Breadcrumb component
interface BreadcrumbItem {
	label: string;
	href?: string;
}

interface BreadcrumbProps {
	items: BreadcrumbItem[];
	className?: string;
}

export const Breadcrumb: FC<BreadcrumbProps> = ({ items, className = "" }) => (
	<nav
		className={`flex items-center gap-1 text-sm ${className}`}
		aria-label="Breadcrumb"
	>
		{items.map((item, index) => (
			<div key={item.label} className="flex items-center gap-1">
				{index > 0 && <ChevronRight className="h-4 w-4 text-gray-600" />}
				{item.href ? (
					<Link
						to={item.href}
						className="text-gray-400 hover:text-white transition-colors"
					>
						{item.label}
					</Link>
				) : (
					<span className="text-gray-300 font-medium">{item.label}</span>
				)}
			</div>
		))}
	</nav>
);
