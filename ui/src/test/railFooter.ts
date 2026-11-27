import type { SidebarFooterProps } from '../ui/Sidebar';

const noop = () => undefined;

/** A complete footer for a SidebarLayout rendered on its own in a test or story. */
export const testFooter: SidebarFooterProps = {
  status: { state: 'connected', label: 'Connected', detail: 'The NIAC daemon is reachable.' },
  help: { label: 'Open help', onClick: noop },
  settings: { label: 'Open settings', onClick: noop },
  theme: { label: 'Switch to light mode', onClick: noop, isDark: true },
  version: { value: '0.0.0', label: 'Version' },
};
