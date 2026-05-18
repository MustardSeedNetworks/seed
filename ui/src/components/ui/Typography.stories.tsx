import type { Meta, StoryFn, StoryObj } from '@storybook/react-vite';
import type React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { AccentLink, Caption, H1, H2, H3, H4, P, SmallText } from './Typography';

const meta: Meta<typeof H1> = {
  title: 'UI/Typography',
  component: H1,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Canonical heading and body text primitives. Use these instead of ad-hoc class strings so the app stays visually consistent.',
      },
    },
  },
  tags: ['autodocs'],
  decorators: [
    (StoryComponent: StoryFn): React.ReactElement => (
      <MemoryRouter>
        <div class="w-[480px]">
          <StoryComponent />
        </div>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Heading1: Story = {
  render: () => <H1>The Seed</H1>,
};

export const Heading2: Story = {
  render: () => <H2>Network diagnostics</H2>,
};

export const Heading3: Story = {
  render: () => <H3>DHCP lease summary</H3>,
};

export const Heading4: Story = {
  render: () => <H4>Interface eth0</H4>,
};

export const Paragraph: Story = {
  render: () => (
    <P>
      The Roots module analyses path quality across the discovered topology using TCP, UDP, and ICMP
      probes. Results are persisted to SQLite and exposed via the REST API.
    </P>
  ),
};

export const SmallTextStory: Story = {
  name: 'SmallText',
  render: () => <SmallText>Last scan: 12 seconds ago</SmallText>,
};

export const CaptionStory: Story = {
  name: 'Caption',
  render: () => <Caption>Sourced from RFC 2544 baseline measurements.</Caption>,
};

export const LinkVariants: Story = {
  name: 'AccentLink',
  render: () => (
    <div class="space-y-3">
      <div>
        <AccentLink href="https://example.com">External link (href)</AccentLink>
      </div>
      <div>
        <AccentLink to="/roots">Internal link (react-router)</AccentLink>
      </div>
      <div>
        <AccentLink onClick={() => undefined}>Button-styled link</AccentLink>
      </div>
    </div>
  ),
};

export const Hierarchy: Story = {
  render: () => (
    <div class="space-y-4">
      <H1>The Seed</H1>
      <H2>Modules</H2>
      <P>Five modules cover the full network diagnostics surface:</P>
      <div class="space-y-2">
        <H3>Roots — path analysis</H3>
        <P>Hop-by-hop latency, jitter, and loss measurements over TCP/UDP/ICMP.</P>
        <SmallText>RFC 2544 baseline supported.</SmallText>
      </div>
      <div class="space-y-2">
        <H3>Canopy — Wi-Fi planning</H3>
        <P>RF coverage and capacity planning informed by IEEE 802.11 telemetry.</P>
      </div>
      <Caption>Last updated 5 minutes ago.</Caption>
    </div>
  ),
};
