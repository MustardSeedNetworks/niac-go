import type { Action } from '../contexts/permissions';
import { useActionPermission } from '../contexts/ScopeContext';
import { Button, type ButtonProps } from '../shared/Button';
import { Tooltip } from './Tooltip';

type ActionButtonProps = ButtonProps & { action?: Action | undefined };

/**
 * The shared Button, gated on the token's scope, with a reason when it is
 * unavailable. The shared Button has no permission hook by design: a product
 * decides at the call site. This is that call site, once: a denied scope
 * disables the button and the tooltip says why, so a read-only token never
 * sees a control that would 403. With no `action` the button is not gated, and
 * with a `title` it explains itself the same way.
 *
 * An unavailable button that has something to say stays in the tab order as
 * `aria-disabled` (the shared Button styles it like `disabled`) and swallows
 * the click, so a keyboard user can still read the reason. One with nothing to
 * say is natively disabled.
 *
 * The explanation is the local Tooltip, not the shared one: the shared bubble
 * is `pointer-events-none`, so it cannot be hovered (WCAG 1.4.13), which
 * e2e/tooltip-keyboard.spec.ts asserts.
 */
export function ActionButton({ action, title, disabled, onClick, ...rest }: ActionButtonProps) {
  const permission = useActionPermission(action);
  const hint = permission.title ?? title;
  const unavailable = Boolean(disabled) || permission.disabled;

  if (!hint) {
    return <Button {...rest} disabled={unavailable} {...(onClick ? { onClick } : {})} />;
  }
  return (
    <Tooltip text={hint}>
      {unavailable ? (
        <Button
          {...rest}
          aria-disabled={true}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
          }}
        />
      ) : (
        <Button {...rest} {...(onClick ? { onClick } : {})} />
      )}
    </Tooltip>
  );
}
