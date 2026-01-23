import type { Meta, StoryObj } from '@storybook/react';
import { Tag } from './Tag';

const meta: Meta<typeof Tag> = {
  title: 'UI/Tag',
  component: Tag,
  tags: ['autodocs'],
  argTypes: {
    colorScheme: {
      control: 'select',
      options: ['gray', 'red', 'green', 'blue', 'yellow', 'purple', 'violet', 'cyan'],
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
type Story = StoryObj<typeof Tag>;

export const Default: Story = {
  args: {
    children: 'Tag',
    colorScheme: 'gray',
  },
};

export const AllColors: Story = {
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

export const ProtocolTags: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Tag colorScheme="purple">TCP</Tag>
      <Tag colorScheme="gray">UDP</Tag>
      <Tag colorScheme="blue">ICMP</Tag>
      <Tag colorScheme="yellow">ARP</Tag>
      <Tag colorScheme="green">DNS</Tag>
      <Tag colorScheme="blue">HTTP</Tag>
      <Tag colorScheme="green">HTTPS</Tag>
      <Tag colorScheme="red">SSH</Tag>
    </div>
  ),
};

export const DeviceTypes: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Tag colorScheme="violet">Router</Tag>
      <Tag colorScheme="blue">Switch</Tag>
      <Tag colorScheme="green">Server</Tag>
      <Tag colorScheme="cyan">Access Point</Tag>
      <Tag colorScheme="yellow">Firewall</Tag>
    </div>
  ),
};
