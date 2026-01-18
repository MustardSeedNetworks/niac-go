import type { LucideIcon } from "lucide-react";
import { ChevronLeft, ChevronRight, Menu, Network, X } from "lucide-react";
import {
	createElement,
	type FC,
	type ReactNode,
	useEffect,
	useState,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";

export interface SidebarNavItem {
	path: string;
	label: string;
	icon: LucideIcon;
	badge?: string;
}

export interface SidebarNavGroup {
	label: string;
	items: SidebarNavItem[];
}

interface SidebarProps {
	groups: SidebarNavGroup[];
	version?: string;
	children: ReactNode;
}

// Store collapsed state in localStorage
const STORAGE_KEY = "niac-sidebar-collapsed";

export const SidebarLayout: FC<SidebarProps> = ({
	groups,
	version,
	children,
}) => {
	const location = useLocation();
	const navigate = useNavigate();
	const [collapsed, setCollapsed] = useState(() => {
		const stored = localStorage.getItem(STORAGE_KEY);
		return stored === "true";
	});
	const [mobileOpen, setMobileOpen] = useState(false);

	// Save collapsed state
	useEffect(() => {
		localStorage.setItem(STORAGE_KEY, String(collapsed));
	}, [collapsed]);

	// Close mobile menu on navigation
	useEffect(() => {
		setMobileOpen(false);
	}, []);

	const isActive = (path: string) =>
		location.pathname === path ||
		(path !== "/" && location.pathname.startsWith(path));

	const renderNavItem = (item: SidebarNavItem) => {
		const active = isActive(item.path);
		return (
			<button
				type="button"
				key={item.path}
				onClick={() => navigate(item.path)}
				className={`
          group flex items-center gap-3 w-full px-3 py-2.5 rounded-lg text-sm font-medium
          transition-all duration-200
          ${
						active
							? "bg-gradient-to-r from-violet-600/30 to-violet-500/20 text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.1)]"
							: "text-gray-400 hover:text-white hover:bg-white/5"
					}
        `}
				title={collapsed ? item.label : undefined}
			>
				{createElement(item.icon, {
					className: `h-5 w-5 flex-shrink-0 ${active ? "text-violet-400" : "text-gray-500 group-hover:text-gray-300"}`,
				})}
				{!collapsed && (
					<>
						<span className="flex-1 text-left truncate">{item.label}</span>
						{item.badge && (
							<span
								className={`px-1.5 py-0.5 text-xs rounded font-medium ${
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
					</>
				)}
				{collapsed && item.badge && (
					<span className="absolute left-12 px-1.5 py-0.5 text-xs rounded font-medium bg-violet-500/20 text-violet-300">
						{item.badge}
					</span>
				)}
			</button>
		);
	};

	const sidebarContent = (
		<>
			{/* Logo */}
			<div
				className={`flex items-center ${collapsed ? "justify-center" : "justify-between"} px-3 py-4 border-b border-white/10`}
			>
				<div
					className={`flex items-center gap-2 ${collapsed ? "justify-center" : ""}`}
				>
					<div className="relative flex-shrink-0">
						<div className="h-9 w-9 rounded-lg bg-gradient-to-br from-violet-500 to-violet-700 flex items-center justify-center shadow-lg shadow-violet-500/30">
							<Network className="h-5 w-5 text-white" />
						</div>
						<div className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 rounded-full bg-emerald-500 border-2 border-gray-900" />
					</div>
					{!collapsed && (
						<span className="font-display font-bold text-lg text-white tracking-tight">
							NIAC
						</span>
					)}
				</div>
				{!collapsed && (
					<button
						type="button"
						onClick={() => setCollapsed(true)}
						className="p-1.5 rounded-lg text-gray-500 hover:text-white hover:bg-white/10 transition-colors lg:flex hidden"
						aria-label="Collapse sidebar"
					>
						<ChevronLeft className="h-4 w-4" />
					</button>
				)}
			</div>

			{/* Navigation */}
			<nav className="flex-1 overflow-y-auto py-4 px-2 space-y-6">
				{groups.map((group) => (
					<div key={group.label}>
						{!collapsed && (
							<h3 className="px-3 mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
								{group.label}
							</h3>
						)}
						{collapsed && <div className="h-px bg-white/10 mx-2 mb-2" />}
						<div className="space-y-1">{group.items.map(renderNavItem)}</div>
					</div>
				))}
			</nav>

			{/* Footer */}
			<div
				className={`px-3 py-4 border-t border-white/10 ${collapsed ? "text-center" : ""}`}
			>
				{version && (
					<div
						className={`text-xs font-mono text-gray-500 ${collapsed ? "" : "flex items-center justify-between"}`}
					>
						{!collapsed && <span>Version</span>}
						<span className={collapsed ? "" : "text-gray-400"}>{version}</span>
					</div>
				)}
				{collapsed && (
					<button
						type="button"
						onClick={() => setCollapsed(false)}
						className="mt-2 p-1.5 rounded-lg text-gray-500 hover:text-white hover:bg-white/10 transition-colors"
						aria-label="Expand sidebar"
					>
						<ChevronRight className="h-4 w-4" />
					</button>
				)}
			</div>
		</>
	);

	return (
		<div className="min-h-screen bg-gradient-to-br from-gray-950 via-gray-900 to-gray-950">
			{/* Mobile header */}
			<div className="lg:hidden fixed top-0 left-0 right-0 z-50 flex items-center justify-between px-4 py-3 bg-gray-900/95 backdrop-blur-xl border-b border-white/10">
				<div className="flex items-center gap-2">
					<div className="h-8 w-8 rounded-lg bg-gradient-to-br from-violet-500 to-violet-700 flex items-center justify-center">
						<Network className="h-4 w-4 text-white" />
					</div>
					<span className="font-display font-bold text-white">NIAC</span>
				</div>
				<button
					type="button"
					onClick={() => setMobileOpen(!mobileOpen)}
					className="p-2 rounded-lg text-gray-400 hover:text-white hover:bg-white/10 transition-colors"
					aria-label={mobileOpen ? "Close menu" : "Open menu"}
				>
					{mobileOpen ? (
						<X className="h-5 w-5" />
					) : (
						<Menu className="h-5 w-5" />
					)}
				</button>
			</div>

			{/* Mobile sidebar overlay */}
			{mobileOpen && (
				<button
					type="button"
					className="lg:hidden fixed inset-0 z-40 bg-black/60 backdrop-blur-sm"
					onClick={() => setMobileOpen(false)}
					aria-label="Close menu"
				/>
			)}

			{/* Mobile sidebar */}
			<aside
				className={`
          lg:hidden fixed top-0 left-0 z-50 h-full w-72 bg-gray-900/95 backdrop-blur-xl border-r border-white/10
          transform transition-transform duration-300 ease-in-out
          ${mobileOpen ? "translate-x-0" : "-translate-x-full"}
        `}
			>
				<div className="flex flex-col h-full">{sidebarContent}</div>
			</aside>

			{/* Desktop sidebar */}
			<aside
				className={`
          hidden lg:flex fixed top-0 left-0 z-40 h-full flex-col
          bg-gray-900/80 backdrop-blur-xl border-r border-white/10
          transition-all duration-300 ease-in-out
          ${collapsed ? "w-16" : "w-64"}
        `}
			>
				{sidebarContent}
			</aside>

			{/* Main content */}
			<main
				className={`
          transition-all duration-300 ease-in-out
          pt-16 lg:pt-0
          ${collapsed ? "lg:pl-16" : "lg:pl-64"}
        `}
			>
				<div className="p-4 sm:p-6 lg:p-8">{children}</div>
			</main>
		</div>
	);
};
