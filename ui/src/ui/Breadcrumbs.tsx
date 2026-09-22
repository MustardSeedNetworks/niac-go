import { ChevronRight, Home } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router';
import { iconSizes } from '../constants/sizes';
import { usePages } from '../pageRegistry';

interface BreadcrumbItem {
  label: string;
  path: string;
  /** False for a path segment no route serves, which must not be a link. */
  routable: boolean;
}

/**
 * Path prefixes that group routes without serving one themselves. `/library`
 * is the sidebar group above /library/walks and /library/pcaps; its name is
 * that group's, so the trail and the rail agree.
 */
const GROUP_LABEL_KEYS = {
  '/library': 'groups.library',
} as const;

type GroupPath = keyof typeof GROUP_LABEL_KEYS;

function isGroupPath(path: string): path is GroupPath {
  return Object.hasOwn(GROUP_LABEL_KEYS, path);
}

/**
 * Breadcrumbs — the trail names a route the way the rest of the app does.
 *
 * The labels used to come from a hand-kept map of 11 of the 16 routes, so the
 * five it missed fell through to the raw URL slug, in English only, while the
 * sidebar and the page header showed a translated name for the same route
 * (#2189). Everything routable now comes from the page registry, which is
 * already the one place a route's name is declared.
 */
export const Breadcrumbs: FC = () => {
  const location = useLocation();
  const { t } = useTranslation('pages');
  const pages = usePages();
  const pathSegments = location.pathname.split('/').filter(Boolean);

  if (pathSegments.length === 0) {
    return null;
  }

  const labelFor = new Map(pages.map((page) => [page.path, page.label]));
  const items: BreadcrumbItem[] = [];
  let currentPath = '';

  for (const segment of pathSegments) {
    currentPath += `/${segment}`;
    const registered = labelFor.get(currentPath);
    items.push({
      path: currentPath,
      routable: registered !== undefined,
      // A dynamic segment — a device hostname in /device-config/:hostname —
      // is the operator's own value and is shown as typed, not title-cased
      // into something they never named.
      label: registered ?? (isGroupPath(currentPath) ? t(GROUP_LABEL_KEYS[currentPath]) : segment),
    });
  }

  return (
    <nav
      aria-label="Breadcrumb"
      className="flex items-center gap-tight text-sm text-text-muted mb-content"
    >
      <Link
        to="/"
        className="flex items-center gap-tight hover:text-text-primary transition-colors"
        aria-label="Home"
      >
        <Home className={iconSizes.sm} />
      </Link>
      {items.map((item, index) => (
        <span key={item.path} className="flex items-center gap-tight">
          <ChevronRight className={`${iconSizes.xs} text-text-disabled`} />
          {index === items.length - 1 || !item.routable ? (
            <span
              data-crumb={true}
              className="text-text-primary font-medium"
              {...(index === items.length - 1 ? { 'aria-current': 'page' as const } : {})}
            >
              {item.label}
            </span>
          ) : (
            <Link
              data-crumb={true}
              to={item.path}
              className="hover:text-text-primary transition-colors"
            >
              {item.label}
            </Link>
          )}
        </span>
      ))}
    </nav>
  );
};
