import type { Action } from '../contexts/permissions';
import { useActionPermission } from '../contexts/ScopeContext';
import { Button, type ButtonProps } from '../shared/Button';
import { Tooltip } from '../shared/Tooltip';

type ActionButtonProps = ButtonProps & { action?: Action | undefined };

/**
 * The shared Button, gated on the token's scope. The shared Button has no
 * permission hook by design: a product decides at the call site. This is that
 * call site, once: a denied scope disables the button and the tooltip says why
 * (a disabled button still explains itself), so a read-only token never sees a
 * control that would 403. With no `action` the button is not gated.
 */
export function ActionButton({ action, title, disabled, ...rest }: ActionButtonProps) {
  const permission = useActionPermission(action);
  const hint = permission.title ?? title;
  const button = <Button {...rest} disabled={disabled || permission.disabled} />;
  return hint ? <Tooltip text={hint}>{button}</Tooltip> : button;
}
