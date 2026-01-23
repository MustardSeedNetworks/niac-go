import type { Meta, StoryObj } from '@storybook/react';
import { Plus, RefreshCw, Trash2 } from 'lucide-react';
import { Button } from './Button';

const meta: Meta<typeof Button> = {
  title: 'UI/Button',
  component: Button,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['solid', 'outline', 'ghost', 'secondary'],
    },
    tone: {
      control: 'select',
      options: ['violet', 'red', 'green', 'blue', 'gray'],
    },
    size: {
      control: 'select',
      options: ['xs', 'sm', 'md', 'lg'],
    },
  },
  decorators: [
    (Story) => (
      <div className="p-8 bg-gray-950">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof Button>;

export const Default: Story = {
  args: {
    children: 'Button',
    variant: 'solid',
    tone: 'violet',
  },
};

export const Variants: Story = {
  render: () => (
    <div className="flex gap-3">
      <Button variant="solid" tone="violet">
        Solid
      </Button>
      <Button variant="outline" tone="violet">
        Outline
      </Button>
      <Button variant="ghost" tone="violet">
        Ghost
      </Button>
      <Button variant="secondary" tone="violet">
        Secondary
      </Button>
    </div>
  ),
};

export const Tones: Story = {
  render: () => (
    <div className="flex gap-3">
      <Button tone="violet">Violet</Button>
      <Button tone="red">Red</Button>
      <Button tone="green">Green</Button>
      <Button tone="blue">Blue</Button>
      <Button tone="gray">Gray</Button>
    </div>
  ),
};

export const Sizes: Story = {
  render: () => (
    <div className="flex items-center gap-3">
      <Button size="xs">Extra Small</Button>
      <Button size="sm">Small</Button>
      <Button size="md">Medium</Button>
      <Button size="lg">Large</Button>
    </div>
  ),
};

export const WithIcons: Story = {
  render: () => (
    <div className="flex gap-3">
      <Button leftIcon={<Plus className="h-4 w-4" />} tone="violet">
        Add Device
      </Button>
      <Button leftIcon={<RefreshCw className="h-4 w-4" />} variant="outline">
        Refresh
      </Button>
      <Button leftIcon={<Trash2 className="h-4 w-4" />} tone="red">
        Delete
      </Button>
    </div>
  ),
};

export const Loading: Story = {
  args: {
    children: 'Saving...',
    loading: true,
    tone: 'violet',
  },
};

export const Disabled: Story = {
  args: {
    children: 'Disabled',
    disabled: true,
    tone: 'violet',
  },
};
