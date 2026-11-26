import type { ReactNode, Ref } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router';
import { usePages } from '../pageRegistry';
import type { BreadcrumbItem, BreadcrumbLinkProps } from '../shared/Breadcrumbs';
import { PageHeader } from '../shared/PageHeader';

/**
 * Path prefixes that group routes without serving one themselves. `/library`
 * is the sidebar group above /library/walks and /library/pcaps; its name is
 * that group's, so the trail and the rail agree.
 */
const GROUP_LABEL_KEYS = { '/library': 'groups.library' } as const;

type GroupPath = keyof typeof GROUP_LABEL_KEYS;

function isGroupPath(path: string): path is GroupPath {
  return Object.hasOwn(GROUP_LABEL_KEYS, path);
}

function renderLink({ href, className, children }: BreadcrumbLinkProps): ReactNode {
  return (
    <Link to={href} className={className}>
      {children}
    </Link>
  );
}

/**
 * useBreadcrumbItems builds the trail for the current route: the dashboard,
 * then one crumb per path segment. Labels come from the page registry, which
 * is where a route's name is already declared for the rail and the header
 * (#2189). A dynamic segment (a device hostname) is shown as typed, and a
 * segment no route serves is not a link.
 */
export function useBreadcrumbItems(current?: string): BreadcrumbItem[] {
  const { pathname } = useLocation();
  const { t } = useTranslation('pages');
  const pages = usePages();
  const labelFor = new Map(pages.map((page) => [page.path, page.label]));
  const home = labelFor.get('/') ?? '';
  const segments = pathname.split('/').filter(Boolean);
  if (segments.length === 0) {
    return [{ label: home }];
  }
  const items: BreadcrumbItem[] = [{ label: home, href: '/' }];
  let path = '';
  segments.forEach((segment, index) => {
    path += `/${segment}`;
    const registered = labelFor.get(path);
    const label = registered ?? (isGroupPath(path) ? t(GROUP_LABEL_KEYS[path]) : segment);
    const last = index === segments.length - 1;
    if (last) {
      items.push({ label: current ?? label });
    } else {
      items.push(registered === undefined ? { label } : { label, href: path });
    }
  });
  return items;
}

interface AppPageHeaderProps {
  kicker: string;
  title: string;
  subtitle?: string | undefined;
  onHelp?: (() => void) | undefined;
  titleRef?: Ref<HTMLHeadingElement> | undefined;
  /** Names the last crumb when the URL cannot (the 404). */
  currentLabel?: string | undefined;
}

/** The one title block every routed page and the 404 open with. */
export function AppPageHeader({
  kicker,
  title,
  subtitle,
  onHelp,
  titleRef,
  currentLabel,
}: AppPageHeaderProps) {
  const { t } = useTranslation('common');
  const breadcrumbs = useBreadcrumbItems(currentLabel);
  return (
    <PageHeader
      kicker={kicker}
      title={title}
      breadcrumbs={breadcrumbs}
      breadcrumbsLabel={t('accessibility.breadcrumb')}
      renderBreadcrumbLink={renderLink}
      {...(subtitle ? { subtitle } : {})}
      {...(onHelp ? { help: { label: t('help.openFor', { title }), onClick: onHelp } } : {})}
      {...(titleRef ? { titleRef } : {})}
    />
  );
}
