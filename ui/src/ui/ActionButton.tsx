import type { ButtonHTMLAttributes } from 'react';
import type { Action } from '../contexts/permissions';
import { useActionPermission } from '../contexts/ScopeContext';

interface Props extends ButtonHTMLAttributes<HTMLButtonElement> {
  action: Action;
}

export function ActionButton({ action, disabled, title, ...props }: Props) {
  const permission = useActionPermission(action);
  return (
    <button
      type="button"
      {...props}
      disabled={disabled || permission.disabled}
      title={permission.title ?? title}
    />
  );
}
