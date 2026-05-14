import { ChevronRight, Home } from 'lucide-react';
import type { FC } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { iconSizes } from '../constants/sizes';

interface BreadcrumbItem {
  label: string;
  path: string;
}

const ROUTE_LABELS: Record<string, string> = {
  '/': 'Command Center',
  '/runtime': 'Runtime Control',
  '/devices': 'Devices & Config',
  '/device-config': 'Config Builder',
  '/topology': 'Topology',
  '/analysis': 'Analysis',
  '/automation': 'Automation',
  '/traffic': 'Traffic Injection',
  '/debug': 'Debug Console',
  '/packets': 'Packet Inspector',
  '/templates': 'Templates',
  '/config-diff': 'Config Diff',
  '/pcap-analyzer': 'PCAP Analyzer',
};

export const Breadcrumbs: FC = () => {
  const location = useLocation();
  const pathSegments = location.pathname.split('/').filter(Boolean);

  if (pathSegments.length === 0) {
    return null;
  }

  const items: BreadcrumbItem[] = [];
  let currentPath = '';

  for (const segment of pathSegments) {
    currentPath += `/${segment}`;
    const label = ROUTE_LABELS[currentPath] ?? segment.replace(/-/g, ' ');
    items.push({ label, path: currentPath });
  }

  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-1 text-sm text-gray-400 mb-4">
      <Link
        to="/"
        className="flex items-center gap-1 hover:text-white transition-colors"
        aria-label="Home"
      >
        <Home className={iconSizes.sm} />
      </Link>
      {items.map((item, index) => (
        <span key={item.path} className="flex items-center gap-1">
          <ChevronRight className={`${iconSizes.xs} text-gray-600`} />
          {index === items.length - 1 ? (
            <span className="text-white font-medium capitalize" aria-current="page">
              {item.label}
            </span>
          ) : (
            <Link to={item.path} className="hover:text-white transition-colors capitalize">
              {item.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
};
