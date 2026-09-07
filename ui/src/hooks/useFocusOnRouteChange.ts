import { type RefObject, useEffect, useRef } from 'react';
import { useLocation } from 'react-router';

/**
 * Moves keyboard focus to the new page's heading after a route change.
 *
 * Without it, following a sidebar link leaves focus on the link: a keyboard
 * or screen-reader user is told nothing about the page they just asked for
 * and has to traverse the whole rail again to reach its content.
 *
 * Two things the shape has to respect. The heading is remounted on every
 * navigation (App keys the page subtree on the pathname), so the target must
 * be read from the ref at focus time rather than captured — child refs are
 * attached during commit, before this effect runs, so `ref.current` is
 * already the new node. And the first render is not a navigation: focusing
 * then would yank focus out of whatever the browser restored on a reload.
 */
export function useFocusOnRouteChange(target: RefObject<HTMLElement | null>): void {
  const { pathname } = useLocation();
  const previousPathname = useRef<string | null>(null);

  useEffect(() => {
    if (previousPathname.current !== null && previousPathname.current !== pathname) {
      target.current?.focus();
    }
    previousPathname.current = pathname;
  }, [pathname, target]);
}
