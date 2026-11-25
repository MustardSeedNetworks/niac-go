import { Link } from 'react-router';
import type { ButtonLinkRenderProps } from '../shared/Button';

/** A ButtonLink's `renderLink` for a router route, so the link keeps client-side navigation. */
export function linkTo(to: string, testId?: string) {
  return (props: ButtonLinkRenderProps) => (
    <Link to={to} {...(testId ? { 'data-testid': testId } : {})} {...props} />
  );
}
