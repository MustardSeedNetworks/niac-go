import type { ButtonHTMLAttributes } from 'react';
import type { Action } from '../contexts/permissions';
import { useActionPermission } from '../contexts/ScopeContext';
import { Tooltip } from './Tooltip';

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  action: Action;
}

export function ActionButton({ action, disabled, title, ...props }: Props) {
  const permission = useActionPermission(action);
  return (
    <Tooltip text={permission.title ?? title}>
      <button type="button" {...props} disabled={disabled || permission.disabled} />
    </Tooltip>
  );
}
