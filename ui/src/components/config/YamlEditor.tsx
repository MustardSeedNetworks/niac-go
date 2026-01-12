import { type FC, useRef, useEffect, useCallback, useMemo } from 'react';
import { EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter } from '@codemirror/view';
import { EditorState, Extension } from '@codemirror/state';
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
import { yaml } from '@codemirror/lang-yaml';
import { syntaxHighlighting, HighlightStyle, foldGutter, indentOnInput, bracketMatching } from '@codemirror/language';
import { tags } from '@lezer/highlight';

/**
 * Custom dark theme for CodeMirror that matches the app's design
 */
const niacTheme = EditorView.theme({
  '&': {
    color: '#d4d4d4',
    backgroundColor: 'transparent',
    fontSize: '14px',
    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace',
  },
  '.cm-content': {
    caretColor: '#a78bfa',
    padding: '16px 0',
  },
  '.cm-cursor': {
    borderLeftColor: '#a78bfa',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
    backgroundColor: 'rgba(139, 92, 246, 0.3)',
  },
  '.cm-activeLine': {
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'rgba(255, 255, 255, 0.05)',
  },
  '.cm-gutters': {
    backgroundColor: 'transparent',
    color: '#6b7280',
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
    backgroundColor: 'rgba(139, 92, 246, 0.2)',
    border: 'none',
    color: '#a78bfa',
    padding: '0 4px',
    margin: '0 4px',
  },
});

/**
 * YAML-specific syntax highlighting
 */
const yamlHighlighting = HighlightStyle.define([
  { tag: tags.keyword, color: '#c678dd' },
  { tag: tags.string, color: '#98c379' },
  { tag: tags.number, color: '#d19a66' },
  { tag: tags.bool, color: '#56b6c2' },
  { tag: tags.null, color: '#56b6c2' },
  { tag: tags.propertyName, color: '#e06c75' },
  { tag: tags.comment, color: '#5c6370', fontStyle: 'italic' },
  { tag: tags.punctuation, color: '#abb2bf' },
  { tag: tags.bracket, color: '#abb2bf' },
  { tag: tags.operator, color: '#abb2bf' },
  { tag: tags.meta, color: '#61afef' },
  { tag: tags.atom, color: '#d19a66' },
  { tag: tags.special(tags.variableName), color: '#e06c75' },
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
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);

  // Build extensions based on props
  const extensions = useMemo(() => {
    const exts: Extension[] = [
      yaml(),
      syntaxHighlighting(yamlHighlighting),
      niacTheme,
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
  const handleUpdate = useCallback((update: { state: EditorState; docChanged: boolean }) => {
    if (update.docChanged && onChange) {
      const newValue = update.state.doc.toString();
      onChange(newValue);

      // Basic YAML validation
      if (onValidationError) {
        const errors = validateYaml(newValue);
        onValidationError(errors);
      }
    }
  }, [onChange, onValidationError]);

  // Initialize editor
  useEffect(() => {
    if (!containerRef.current) return;

    // Destroy existing view
    if (viewRef.current) {
      viewRef.current.destroy();
    }

    const state = EditorState.create({
      doc: value,
      extensions: [
        ...extensions,
        EditorView.updateListener.of(handleUpdate),
      ],
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
  }, [extensions, handleUpdate]); // Note: value is intentionally excluded

  // Update content when value prop changes externally
  useEffect(() => {
    if (!viewRef.current) return;

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

  // Container styles
  const containerStyle = {
    height,
    minHeight,
    maxHeight,
    overflow: 'auto' as const,
  };

  return (
    <div
      ref={containerRef}
      className={`rounded-xl border border-white/10 bg-gray-950/70 overflow-hidden ${className}`}
      style={containerStyle}
      aria-label="YAML editor"
    />
  );
};

/**
 * Read-only YAML viewer component (convenience wrapper)
 */
export const YamlViewer: FC<Omit<YamlEditorProps, 'readOnly' | 'onChange'>> = (props) => {
  return <YamlEditor {...props} readOnly />;
};

/**
 * Basic YAML validation
 * Returns array of error messages (empty if valid)
 */
function validateYaml(content: string): string[] {
  const errors: string[] = [];
  const lines = content.split('\n');

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Check for tabs (YAML should use spaces)
    if (line.includes('\t')) {
      errors.push(`Line ${lineNum}: Tabs are not allowed in YAML, use spaces instead`);
    }

    // Check for inconsistent indentation
    const leadingSpaces = line.match(/^(\s*)/)?.[1].length ?? 0;
    if (leadingSpaces > 0 && leadingSpaces % 2 !== 0) {
      errors.push(`Line ${lineNum}: Inconsistent indentation (should be multiple of 2 spaces)`);
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
