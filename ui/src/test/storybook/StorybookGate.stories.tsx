import type { Meta, StoryObj } from '@storybook/react-vite';
import { expect, fn, userEvent, within } from 'storybook/test';
import { Button } from '../../ui/Button';

/**
 * A gate that has not been proven able to fail is not worth trusting.
 *
 * These two stories are the Storybook suite's self-test. Running with
 * VITE_STORYBOOK_INJECT_DEFECT=interaction or =accessibility plants a real
 * defect of that kind, and scripts/test-storybook-contract.sh asserts the suite
 * goes red for the right reason. Without them, a harness that silently stopped
 * running stories would look exactly like a harness finding nothing wrong.
 */
type Defect = 'accessibility' | 'interaction';

interface GateControlsProps {
  defect?: Defect;
  onSave: () => void;
}

function GateControls({ defect, onSave }: GateControlsProps) {
  // aria-hidden on the label leaves a button with no discernible text, which is
  // axe's button-name rule — invisible on screen, fatal to a screen reader.
  const buttonLabel = <span aria-hidden={defect === 'accessibility'}>Save target</span>;

  return (
    <form className="stack" onSubmit={(event) => event.preventDefault()}>
      <Button disabled={defect === 'interaction'} onClick={onSave}>
        {buttonLabel}
      </Button>
    </form>
  );
}

const injectedDefect = import.meta.env.VITE_STORYBOOK_INJECT_DEFECT as Defect | undefined;

const meta = {
  title: 'Test/Storybook gate',
  component: GateControls,
  parameters: {
    a11y: { test: 'error' },
  },
} satisfies Meta<typeof GateControls>;

export default meta;
type Story = StoryObj<typeof meta>;

export const SharedComponentInteraction: Story = {
  args: {
    defect: injectedDefect === 'interaction' ? injectedDefect : undefined,
    onSave: fn(),
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole('button', { name: 'Save target' }));
    await expect(args.onSave).toHaveBeenCalledOnce();
  },
};

export const SharedComponentAccessibility: Story = {
  args: {
    defect: injectedDefect === 'accessibility' ? injectedDefect : undefined,
    onSave: fn(),
  },
};
