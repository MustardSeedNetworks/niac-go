import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { yaml } from '@codemirror/lang-yaml';
import {
  bracketMatching,
  foldGutter,
  HighlightStyle,
  indentOnInput,
  syntaxHighlighting,
} from '@codemirror/language';
import { EditorState, type Extension, StateEffect, StateField } from '@codemirror/state';
import {
  Decoration,
  type DecorationSet,
  EditorView,
  highlightActiveLine,
  highlightActiveLineGutter,
  keymap,
  lineNumbers,
} from '@codemirror/view';
import { tags } from '@lezer/highlight';
import { type FC, useCallback, useEffect, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import i18n from '../../i18n';

/**
 * Line decoration used to highlight the offending line reported by a
 * structured YAML parse error (see internal/api ErrorDetail.line). Set via
 * `setErrorLineEffect` and cleared by dispatching it with `line: null`.
 */
const setErrorLineEffect = StateEffect.define<number | null>();

const errorLineDecoration = Decoration.line({ class: 'cm-niac-error-line' });

const errorLineField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(decorations, tr) {
    let next = decorations.map(tr.changes);

    for (const effect of tr.effects) {
      if (effect.is(setErrorLineEffect)) {
        if (effect.value === null) {
          next = Decoration.none;
        } else {
          const lineNum = Math.min(Math.max(effect.value, 1), tr.state.doc.lines);
          const line = tr.state.doc.line(lineNum);
          next = Decoration.set([errorLineDecoration.range(line.from)]);
        }
      }
    }

    return next;
  },
  provide: (field) => EditorView.decorations.from(field),
});

const errorLineTheme = EditorView.theme({
  '.cm-niac-error-line': {
    backgroundColor: 'color-mix(in srgb, var(--color-status-error) 18%, transparent)',
    borderLeft: '3px solid var(--color-status-error)',
  },
});

/**
 * CodeMirror theme, painted from the app's design tokens.
 *
 * CodeMirror styles from JS rather than from utility classes, so the tokens
 * are referenced as `var(--color-*)` directly. That is also what makes the
 * editor follow the theme: the custom properties are redefined under `.dark`
 * (ui/src/theme/), so light and dark need no JS branch here.
 */
const niacTheme = EditorView.theme({
  '&': {
    color: 'var(--color-text-primary)',
    backgroundColor: 'transparent',
    fontSize: '14px',
    fontFamily:
      'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace',
  },
  '.cm-content': {
    caretColor: 'var(--color-brand-primary)',
    padding: '16px 0',
  },
  '.cm-cursor': {
    borderLeftColor: 'var(--color-brand-primary)',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
    backgroundColor: 'color-mix(in srgb, var(--color-brand-primary) 30%, transparent)',
  },
  '.cm-activeLine': {
    backgroundColor: 'color-mix(in srgb, var(--color-text-primary) 5%, transparent)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'color-mix(in srgb, var(--color-text-primary) 5%, transparent)',
  },
  '.cm-gutters': {
    backgroundColor: 'transparent',
    color: 'var(--color-text-muted)',
    border: 'none',
    paddingRight: '8px',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    padding: '0 8px 0 16px',
    minWidth: '40px',
  },
  '.cm-foldGutter .cm-gutterElement': {
    padding: '0 4px',
    cursor: 'pointer',
  },
  '.cm-foldPlaceholder': {
    backgroundColor: 'color-mix(in srgb, var(--color-brand-primary) 20%, transparent)',
    border: 'none',
    color: 'var(--color-brand-primary)',
    padding: '0 4px',
    margin: '0 4px',
  },
});

/**
 * YAML-specific syntax highlighting
 */
const yamlHighlighting = HighlightStyle.define([
  { tag: tags.keyword, color: 'var(--color-syntax-keyword)' },
  { tag: tags.string, color: 'var(--color-syntax-string)' },
  { tag: tags.number, color: 'var(--color-syntax-number)' },
  { tag: tags.bool, color: 'var(--color-syntax-atom)' },
  { tag: tags.null, color: 'var(--color-syntax-atom)' },
  { tag: tags.propertyName, color: 'var(--color-syntax-property)' },
  { tag: tags.comment, color: 'var(--color-syntax-comment)', fontStyle: 'italic' },
  { tag: tags.punctuation, color: 'var(--color-syntax-punctuation)' },
  { tag: tags.bracket, color: 'var(--color-syntax-punctuation)' },
  { tag: tags.operator, color: 'var(--color-syntax-punctuation)' },
  { tag: tags.meta, color: 'var(--color-syntax-meta)' },
  { tag: tags.atom, color: 'var(--color-syntax-number)' },
  { tag: tags.special(tags.variableName), color: 'var(--color-syntax-property)' },
]);

interface YamlEditorProps {
  /** The YAML content to display/edit */
  value: string;
  /** Called when the content changes (if not readOnly) */
  onChange?: (value: string) => void;
  /** Whether the editor is read-only */
  readOnly?: boolean;
  /** Placeholder text when empty */
  placeholder?: string;
  /** Height of the editor (CSS value) */
  height?: string;
  /** Minimum height of the editor (CSS value) */
  minHeight?: string;
  /** Maximum height of the editor (CSS value) */
  maxHeight?: string;
  /** Whether to show line numbers */
  showLineNumbers?: boolean;
  /** Whether to show fold gutters */
  showFoldGutter?: boolean;
  /** Whether to wrap long lines */
  lineWrapping?: boolean;
  /** Additional CSS class names */
  className?: string;
  /** Callback when validation errors are detected */
  onValidationError?: (errors: string[]) => void;
  /** 1-based line to highlight and scroll into view (e.g. a structured YAML
   * parse error's line number). `null`/`undefined` clears the highlight. */
  errorLine?: number | null;
}

/**
 * YamlEditor - A CodeMirror-based YAML editor with syntax highlighting
 *
 * Features:
 * - Full YAML syntax highlighting
 * - Line numbers and fold gutters
 * - Read-only mode for viewing
 * - Dark theme matching the app design
 * - Optional line wrapping
 * - Validation callbacks
 */
export const YamlEditor: FC<YamlEditorProps> = ({
  value,
  onChange,
  readOnly = false,
  placeholder,
  height = 'auto',
  minHeight = '200px',
  maxHeight = '500px',
  showLineNumbers = true,
  showFoldGutter = true,
  lineWrapping = false,
  className = '',
  onValidationError,
  errorLine,
}) => {
  const { t } = useTranslation('pages');
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  // Build extensions based on props
  const extensions = useMemo(() => {
    const exts: Extension[] = [
      yaml(),
      syntaxHighlighting(yamlHighlighting),
      niacTheme,
      errorLineField,
      errorLineTheme,
      history(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      indentOnInput(),
      bracketMatching(),
      ...(lineWrapping ? [EditorView.lineWrapping] : []),
    ];

    if (showLineNumbers) {
      exts.push(lineNumbers());
      exts.push(highlightActiveLineGutter());
    }

    if (showFoldGutter) {
      exts.push(foldGutter());
    }

    if (!readOnly) {
      exts.push(highlightActiveLine());
    }

    if (readOnly) {
      exts.push(EditorState.readOnly.of(true));
      exts.push(EditorView.editable.of(false));
    }

    if (placeholder) {
      exts.push(EditorView.contentAttributes.of({ 'aria-placeholder': placeholder }));
    }

    return exts;
  }, [readOnly, placeholder, showLineNumbers, showFoldGutter, lineWrapping]);

  // Handle content updates
  const handleUpdate = useCallback(
    (update: { state: EditorState; docChanged: boolean }) => {
      if (update.docChanged && onChange) {
        const newValue = update.state.doc.toString();
        onChange(newValue);

        // Basic YAML validation
        if (onValidationError) {
          const errors = validateYaml(newValue);
          onValidationError(errors);
        }
      }
    },
    [onChange, onValidationError],
  );

  // Initialize editor. `value` is intentionally excluded from the deps
  // array: it seeds the initial doc only. Re-running this effect on every
  // keystroke would destroy and recreate the whole CodeMirror view (losing
  // cursor position, focus, and undo history) each time `onChange` flows a
  // new value back in as a prop. External value changes (load, reset,
  // discard) are instead applied by the sync effect below, which dispatches
  // a targeted doc replacement instead of a full teardown.
  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    // Destroy existing view
    if (viewRef.current) {
      viewRef.current.destroy();
    }

    const state = EditorState.create({
      doc: value,
      extensions: [...extensions, EditorView.updateListener.of(handleUpdate)],
    });

    const view = new EditorView({
      state,
      parent: containerRef.current,
    });

    viewRef.current = view;

    return () => {
      view.destroy();
      viewRef.current = null;
    };
  }, [extensions, handleUpdate]);

  // Update content when value prop changes externally
  useEffect(() => {
    if (!viewRef.current) {
      return;
    }

    const currentValue = viewRef.current.state.doc.toString();
    if (currentValue !== value) {
      viewRef.current.dispatch({
        changes: {
          from: 0,
          to: currentValue.length,
          insert: value,
        },
      });
    }
  }, [value]);

  // Highlight and scroll to the reported error line, when set.
  useEffect(() => {
    const view = viewRef.current;
    if (!view) {
      return;
    }

    if (errorLine === undefined || errorLine === null || errorLine < 1) {
      view.dispatch({ effects: setErrorLineEffect.of(null) });
      return;
    }

    const lineNum = Math.min(errorLine, view.state.doc.lines);
    const line = view.state.doc.line(lineNum);

    view.dispatch({
      effects: [
        setErrorLineEffect.of(errorLine),
        EditorView.scrollIntoView(line.from, { y: 'center' }),
      ],
    });
  }, [errorLine]);

  // Container styles
  const containerStyle = {
    height,
    minHeight,
    maxHeight,
    overflow: 'auto' as const,
  };

  return (
    <section
      ref={containerRef}
      className={`rounded-xl border border-surface-border bg-bg-base/70 overflow-hidden ${className}`}
      style={containerStyle}
      aria-label={t('configDiff.yamlEditorLabel')}
    />
  );
};

/**
 * Read-only YAML viewer component (convenience wrapper)
 */
export const YamlViewer: FC<Omit<YamlEditorProps, 'readOnly' | 'onChange'>> = (props) => (
  <YamlEditor {...props} readOnly={true} />
);

/**
 * Basic YAML validation
 * Returns array of error messages (empty if valid)
 */
function validateYaml(content: string): string[] {
  const errors: string[] = [];
  const lines = content.split('\n');

  for (const [i, line] of lines.entries()) {
    const lineNum = i + 1;

    // Check for tabs (YAML should use spaces)
    if (line.includes('\t')) {
      errors.push(i18n.t('configDiff.yamlTabsError', { ns: 'pages', lineNum }));
    }

    // Check for inconsistent indentation
    const leadingSpaces = line.match(/^(\s*)/)?.[1]?.length ?? 0;
    if (leadingSpaces > 0 && leadingSpaces % 2 !== 0) {
      errors.push(i18n.t('configDiff.yamlIndentationError', { ns: 'pages', lineNum }));
    }

    // Check for trailing colons without value on same line
    if (line.match(/:\s*#/) && !line.match(/:\s+\S/)) {
      // This is likely a comment after key:, which is fine
    }

    // Check for duplicate keys at same level (simplified check)
    // Note: Full duplicate key detection would require parsing
  }

  return errors;
}

export default YamlEditor;
