/**
 * YamlEditor.test.tsx — Phase 5d error-line highlighting.
 *
 * Backend YAML parse errors (internal/api/yaml_errors.go) and the client
 * `yaml` parser both surface a 1-based line number; the editor must
 * highlight that line via CodeMirror's decoration API (`errorLine` prop)
 * so the operator can find the problem without counting lines by hand.
 */
import { undo } from '@codemirror/commands';
import { EditorView } from '@codemirror/view';
import { act, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import '../../i18n';
import { YamlEditor } from './YamlEditor';

const CONTENT = 'devices:\n  - name: r1\n    type: router\n  - name: r2\n';

describe('YamlEditor callback updates', () => {
  it('syncs externally selected content without reporting a user edit', () => {
    const onChange = vi.fn();
    const onValidationError = vi.fn();
    const { rerender } = render(
      <YamlEditor
        ariaLabel="Test YAML editor"
        value={CONTENT}
        onChange={onChange}
        onValidationError={onValidationError}
      />,
    );
    const textbox = screen.getByRole('textbox', { name: 'Test YAML editor' });
    const view = EditorView.findFromDOM(textbox);
    if (!view) throw new Error('YAML editor did not mount');
    const selected = 'name: r2\ntype: router\n';
    rerender(
      <YamlEditor
        ariaLabel="Test YAML editor"
        value={selected}
        onChange={onChange}
        onValidationError={onValidationError}
      />,
    );
    expect(screen.getByRole('textbox', { name: 'Test YAML editor' })).toBe(textbox);
    expect(view.state.doc.toString()).toBe(selected);
    expect(onChange).not.toHaveBeenCalled();
    expect(onValidationError).not.toHaveBeenCalled();

    act(() => view.dispatch({ changes: { from: selected.length, insert: '# user edit' } }));
    expect(onChange).toHaveBeenCalledExactlyOnceWith(`${selected}# user edit`);
    expect(onValidationError).toHaveBeenCalledExactlyOnceWith([]);
  });

  it('preserves the editor and undo history while calling the latest change handler', () => {
    const firstChange = vi.fn();
    const nextChange = vi.fn();
    const { rerender } = render(
      <YamlEditor ariaLabel="Test YAML editor" value={CONTENT} onChange={firstChange} />,
    );
    const textbox = screen.getByRole('textbox', { name: 'Test YAML editor' });
    const view = EditorView.findFromDOM(textbox);
    if (!view) throw new Error('YAML editor did not mount');
    act(() => view.dispatch({ changes: { from: CONTENT.length, insert: '# edit' } }));
    expect(firstChange).toHaveBeenLastCalledWith(`${CONTENT}# edit`);

    rerender(
      <YamlEditor ariaLabel="Test YAML editor" value={`${CONTENT}# edit`} onChange={nextChange} />,
    );
    expect(screen.getByRole('textbox', { name: 'Test YAML editor' })).toBe(textbox);
    act(() => {
      expect(undo(view)).toBe(true);
    });
    expect(nextChange).toHaveBeenLastCalledWith(CONTENT);
    expect(firstChange).toHaveBeenCalledTimes(1);
  });
});

describe('YamlEditor — error line highlighting', () => {
  it('applies the error-line decoration class to the reported line', () => {
    const { container } = render(
      <YamlEditor ariaLabel="Test YAML editor" value={CONTENT} errorLine={3} />,
    );

    const highlighted = container.querySelectorAll('.cm-niac-error-line');
    expect(highlighted).toHaveLength(1);
  });

  it('renders no highlight when errorLine is not set', () => {
    const { container } = render(<YamlEditor ariaLabel="Test YAML editor" value={CONTENT} />);

    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(0);
  });

  it('clamps an out-of-range line to the last line instead of throwing', () => {
    const { container } = render(
      <YamlEditor ariaLabel="Test YAML editor" value={CONTENT} errorLine={999} />,
    );

    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(1);
  });

  it('clears the highlight when errorLine is reset to null', () => {
    const { container, rerender } = render(
      <YamlEditor ariaLabel="Test YAML editor" value={CONTENT} errorLine={2} />,
    );
    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(1);

    rerender(<YamlEditor ariaLabel="Test YAML editor" value={CONTENT} errorLine={null} />);
    expect(container.querySelectorAll('.cm-niac-error-line')).toHaveLength(0);
  });
});
