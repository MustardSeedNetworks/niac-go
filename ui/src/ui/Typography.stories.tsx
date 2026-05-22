/**
 * Typography primitive stories (Wave 5 / niac-W5-2b).
 *
 * Showcases the typography scale (H1-H4, P, SmallText, Caption) plus
 * the AccentLink helper.
 */
import type { Meta, StoryObj } from '@storybook/react-vite';
import { AccentLink, Caption, H1, H2, H3, H4, P, SmallText } from './Typography';

const meta: Meta = {
  title: 'UI/Typography',
  parameters: { layout: 'padded' },
};
export default meta;

type Story = StoryObj;

export const Scale: Story = {
  render: () => (
    <div className="space-y-3 max-w-xl">
      <H1>H1 — Heading One</H1>
      <H2>H2 — Heading Two</H2>
      <H3>H3 — Heading Three</H3>
      <H4>H4 — Heading Four</H4>
      <P>P — Body paragraph. The quick brown fox jumps over the lazy dog.</P>
      <SmallText>SmallText — secondary helper line.</SmallText>
      <Caption>Caption — finest print, often used for labels.</Caption>
    </div>
  ),
};

export const AccentLinkExample: Story = {
  name: 'AccentLink',
  render: () => (
    <P>
      Visit the <AccentLink href="https://mustardseed.io">mustardseed.io</AccentLink> docs for more
      detail.
    </P>
  ),
};
