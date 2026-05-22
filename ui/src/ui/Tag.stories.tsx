/**
 * Tag primitive stories (Wave 5 / #636).
 *
 * Full colorScheme matrix.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { Tag } from './Tag';

const meta: Meta<typeof Tag> = {
  title: 'UI/Tag',
  component: Tag,
  parameters: { layout: 'centered' },
  argTypes: {
    colorScheme: {
      control: 'select',
      options: ['gray', 'red', 'green', 'blue', 'yellow', 'purple', 'violet', 'cyan'],
    },
  },
};
export default meta;

type Story = StoryObj<typeof Tag>;

export const Default: Story = { args: { children: 'Default', colorScheme: 'gray' } };
export const Green: Story = { args: { children: 'Active', colorScheme: 'green' } };
export const Red: Story = { args: { children: 'Error', colorScheme: 'red' } };
export const Blue: Story = { args: { children: 'Info', colorScheme: 'blue' } };
export const Yellow: Story = { args: { children: 'Warning', colorScheme: 'yellow' } };
export const Violet: Story = { args: { children: 'Beta', colorScheme: 'violet' } };

export const ColorMatrix: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Tag colorScheme="gray">Gray</Tag>
      <Tag colorScheme="red">Red</Tag>
      <Tag colorScheme="green">Green</Tag>
      <Tag colorScheme="blue">Blue</Tag>
      <Tag colorScheme="yellow">Yellow</Tag>
      <Tag colorScheme="purple">Purple</Tag>
      <Tag colorScheme="violet">Violet</Tag>
      <Tag colorScheme="cyan">Cyan</Tag>
    </div>
  ),
};
