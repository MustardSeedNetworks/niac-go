import { describe, expect, it } from 'vitest';
import { can } from './permissions';

describe('action permissions', () => {
  it.each(['view', 'filter', 'export'] as const)('allows viewers to %s', (action) => {
    expect(can('read-only', action)).toBe(true);
  });
  it.each(['start', 'stop', 'edit', 'upload', 'inject', 'delete'] as const)(
    'requires an operator for %s',
    (action) => {
      expect(can(null, action)).toBe(false);
      expect(can('read-only', action)).toBe(false);
      expect(can('read-write', action)).toBe(true);
      expect(can('admin', action)).toBe(true);
    },
  );
  it('reserves whole-config replacement and bundle install for admins', () => {
    expect(can('read-write', 'admin')).toBe(false);
    expect(can('admin', 'admin')).toBe(true);
    expect(can(null, 'view')).toBe(false);
  });
});
