export type Scope = 'read-only' | 'read-write' | 'admin';
export type Action =
  | 'view'
  | 'filter'
  | 'export'
  | 'start'
  | 'stop'
  | 'edit'
  | 'upload'
  | 'inject'
  | 'delete'
  | 'admin';

export function can(scope: Scope | null, action: Action): boolean {
  if (scope === null) return false;
  if (action === 'view' || action === 'filter' || action === 'export') return true;
  if (action === 'admin') return scope === 'admin';
  return scope === 'read-write' || scope === 'admin';
}
