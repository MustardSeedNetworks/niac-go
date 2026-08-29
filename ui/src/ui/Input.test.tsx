/**
 * Tests for the shared form primitives.
 *
 * Each component derives its element id from the label when no id is given, and
 * that id is what ties the <label> to its control. A broken derivation is
 * invisible on screen and breaks every screen reader and every label-based
 * query, so it is asserted directly rather than through appearance.
 */

import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  Checkbox,
  FormGroup,
  FormSection,
  Input,
  SearchInput,
  Select,
  Textarea,
  Toggle,
} from './Input';

describe('Input', () => {
  it('derives an id from the label and associates the two', () => {
    render(<Input label="Device Name" />);

    const field = screen.getByLabelText('Device Name');
    expect(field.id).toBe('device-name');
  });

  it('prefers an explicit id over the derived one', () => {
    render(<Input label="Device Name" id="explicit" />);

    expect(screen.getByLabelText('Device Name').id).toBe('explicit');
  });

  it('renders without a label', () => {
    render(<Input placeholder="unlabelled" />);

    expect(screen.getByPlaceholderText('unlabelled')).toBeDefined();
    expect(screen.queryByRole('label')).toBeNull();
  });

  it('shows the hint when there is no error', () => {
    render(<Input label="Host" hint="A hostname or IP" />);

    expect(screen.getByText('A hostname or IP')).toBeDefined();
  });

  it('shows the error instead of the hint when both are given', () => {
    // Error must win: showing the hint would hide why the field is red.
    render(<Input label="Host" hint="A hostname or IP" error="Required" />);

    expect(screen.getByText('Required')).toBeDefined();
    expect(screen.queryByText('A hostname or IP')).toBeNull();
  });

  it('renders neither line when there is no error and no hint', () => {
    const { container } = render(<Input label="Host" />);

    expect(container.querySelector('p')).toBeNull();
  });

  it('renders left and right icons', () => {
    render(
      <Input
        label="Host"
        leftIcon={<span data-testid="left" />}
        rightIcon={<span data-testid="right" />}
      />,
    );

    expect(screen.getByTestId('left')).toBeDefined();
    expect(screen.getByTestId('right')).toBeDefined();
  });

  it('forwards the change handler and arbitrary input props', () => {
    const onChange = vi.fn();
    render(<Input label="Host" onChange={onChange} disabled={true} />);

    const field = screen.getByLabelText('Host') as HTMLInputElement;
    expect(field.disabled).toBe(true);

    fireEvent.change(field, { target: { value: 'x' } });
    expect(onChange).toHaveBeenCalled();
  });
});

describe('Textarea', () => {
  it('derives an id from the label', () => {
    render(<Textarea label="Notes Field" />);

    expect(screen.getByLabelText('Notes Field').id).toBe('notes-field');
  });

  it('shows the error in preference to the hint', () => {
    render(<Textarea label="Notes" hint="optional" error="too long" />);

    expect(screen.getByText('too long')).toBeDefined();
    expect(screen.queryByText('optional')).toBeNull();
  });
});

describe('Select', () => {
  const options = [
    { value: 'a', label: 'Alpha' },
    { value: 'b', label: 'Beta', disabled: true },
  ];

  it('renders every option', () => {
    render(<Select label="Mode" options={options} />);

    expect(screen.getByRole('option', { name: 'Alpha' })).toBeDefined();
    expect((screen.getByRole('option', { name: 'Beta' }) as HTMLOptionElement).disabled).toBe(true);
  });

  it('renders a disabled placeholder when one is given', () => {
    render(<Select label="Mode" options={options} placeholder="Choose…" />);

    const placeholder = screen.getByRole('option', { name: 'Choose…' }) as HTMLOptionElement;
    expect(placeholder.disabled).toBe(true);
    expect(placeholder.value).toBe('');
  });

  it('omits the placeholder option when none is given', () => {
    render(<Select label="Mode" options={options} />);

    expect(screen.getAllByRole('option')).toHaveLength(2);
  });

  it('reports the selected value, not the event', () => {
    // The wrapper unwraps e.target.value; a caller receiving the event instead
    // would silently store "[object Object]".
    const onChange = vi.fn();
    render(<Select label="Mode" options={options} onChange={onChange} />);

    fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'a' } });
    expect(onChange).toHaveBeenCalledWith('a');
  });

  it('does not throw when no onChange is supplied', () => {
    render(<Select label="Mode" options={options} />);

    expect(() =>
      fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'a' } }),
    ).not.toThrow();
  });

  it('shows an error message', () => {
    render(<Select label="Mode" options={options} error="pick one" />);

    expect(screen.getByText('pick one')).toBeDefined();
  });
});

describe('Checkbox', () => {
  it('derives an id from the label and associates the two', () => {
    render(<Checkbox label="Enable SNMP" />);

    expect(screen.getByLabelText('Enable SNMP').id).toBe('enable-snmp');
  });

  it('renders an optional description', () => {
    render(<Checkbox label="Enable SNMP" description="Responds to walks" />);

    expect(screen.getByText('Responds to walks')).toBeDefined();
  });

  it('omits the description element when there is none', () => {
    const { container } = render(<Checkbox label="Enable SNMP" />);

    expect(container.querySelector('p')).toBeNull();
  });
});

describe('Toggle', () => {
  it('exposes its state through role=switch', () => {
    render(<Toggle label="Babble" checked={true} readOnly={true} />);

    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('true');
  });

  it('reports unchecked state', () => {
    render(<Toggle label="Babble" checked={false} readOnly={true} />);

    expect(screen.getByRole('switch').getAttribute('aria-checked')).toBe('false');
  });

  it('clicking the switch toggles the underlying checkbox', () => {
    const onChange = vi.fn();
    render(<Toggle label="Babble" checked={false} onChange={onChange} />);

    fireEvent.click(screen.getByRole('switch'));
    expect(onChange).toHaveBeenCalled();
  });

  it('renders an optional description', () => {
    render(
      <Toggle label="Babble" description="Emit stray traffic" checked={false} readOnly={true} />,
    );

    expect(screen.getByText('Emit stray traffic')).toBeDefined();
  });
});

describe('SearchInput', () => {
  it('hides the clear button until there is a value', () => {
    const { rerender } = render(<SearchInput value="" readOnly={true} />);
    expect(screen.queryByRole('button')).toBeNull();

    rerender(<SearchInput value="dev" readOnly={true} />);
    expect(screen.getByRole('button')).toBeDefined();
  });

  it('hides the clear button when value is undefined', () => {
    render(<SearchInput readOnly={true} />);

    expect(screen.queryByRole('button')).toBeNull();
  });

  it('calls onClear when one is supplied', () => {
    const onClear = vi.fn();
    const onChange = vi.fn();
    render(<SearchInput value="dev" onClear={onClear} onChange={onChange} />);

    fireEvent.click(screen.getByRole('button'));
    expect(onClear).toHaveBeenCalled();
    // onClear takes precedence; firing both would clear twice.
    expect(onChange).not.toHaveBeenCalled();
  });

  it('falls back to a synthetic empty change when there is no onClear', () => {
    const onChange = vi.fn();
    render(<SearchInput value="dev" onChange={onChange} />);

    fireEvent.click(screen.getByRole('button'));
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ target: expect.objectContaining({ value: '' }) }),
    );
  });

  it('does not throw when neither handler is supplied', () => {
    render(<SearchInput value="dev" readOnly={true} />);

    expect(() => fireEvent.click(screen.getByRole('button'))).not.toThrow();
  });
});

describe('FormSection and FormGroup', () => {
  it('renders the title, description and children', () => {
    render(
      <FormSection title="SNMP" description="Agent settings">
        <span data-testid="child" />
      </FormSection>,
    );

    expect(screen.getByRole('heading', { name: 'SNMP' })).toBeDefined();
    expect(screen.getByText('Agent settings')).toBeDefined();
    expect(screen.getByTestId('child')).toBeDefined();
  });

  it('omits the description when there is none', () => {
    const { container } = render(<FormSection title="SNMP">{null}</FormSection>);

    expect(container.querySelector('p')).toBeNull();
  });

  it('FormGroup renders its children', () => {
    render(
      <FormGroup>
        <span data-testid="grouped" />
      </FormGroup>,
    );

    expect(screen.getByTestId('grouped')).toBeDefined();
  });
});
