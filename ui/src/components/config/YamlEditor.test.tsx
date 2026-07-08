/**
 * YamlEditor.test.tsx — Phase 5d error-line highlighting.
 *
 * Backend YAML parse errors (internal/api/yaml_errors.go) and the client
 * `yaml` parser both surface a 1-based line number; the editor must
 * highlight that line via CodeMirror's decoration API (`errorLine` prop)
 * so the operator can find the problem without counting lines by hand.
 */
import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import '../../i18n';
import { YamlEditor } from './YamlEditor';

const CONTENT = 'devices:\n  - name: r1\n    type: router\n  - name: r2\n';

describe('YamlEditor — error line highlighting', () => {
  it('applies the error-line decoration class to the reported line', () => {
    const { container } = render(<YamlEditor value={CONTENT} errorLine={3} />);

    const highlighted = container.querySelectorAll('.cm-niac-error-line');
    expect(highlighted).toHaveLength(1);
  });

  it('renders no highlight when errorLine is not set', () => {
    const { container } = render(<YamlEditor value={CONTENT} />);

    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(0);
  });

  it('clamps an out-of-range line to the last line instead of throwing', () => {
    const { container } = render(<YamlEditor value={CONTENT} errorLine={999} />);

    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(1);
  });

  it('clears the highlight when errorLine is reset to null', () => {
    const { container, rerender } = render(<YamlEditor value={CONTENT} errorLine={2} />);
    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(1);

    rerender(<YamlEditor value={CONTENT} errorLine={null} />);
    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(0);
  });
});
