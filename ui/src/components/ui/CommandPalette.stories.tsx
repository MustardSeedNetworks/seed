import type { Decorator, Meta, StoryObj } from '@storybook/react-vite';
import { Activity, BookOpen, Home, Sparkles } from 'lucide-react';
import { useState } from 'react';
import { StoryRouter } from '../../test/StoryRouter';
import type { SidebarNavGroup } from '../../ui/Sidebar';
import { Button } from './Button';
import { CommandPalette } from './CommandPalette';

const navGroups: SidebarNavGroup[] = [
  {
    label: 'Diagnostics',
    items: [
      { label: 'Path Analysis', path: '/path', icon: Home },
      { label: 'Wi-Fi', path: '/wifi', icon: Activity },
      { label: 'Link', path: '/link', icon: Sparkles },
    ],
  },
  {
    label: 'Help',
    items: [{ label: 'Documentation', path: '/help', icon: BookOpen }],
  },
];

const meta: Meta<typeof CommandPalette> = {
  title: 'UI/CommandPalette',
  component: CommandPalette,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Keyboard-first command/navigation palette (cmdk). Opens on Cmd+K (macOS) / Ctrl+K (others). Populates with sidebar nav entries plus common actions.',
      },
    },
  },
  tags: ['autodocs'],
  decorators: [
    (Story: Parameters<Decorator>[0]) => (
      <StoryRouter>
        <div className="min-h-[60vh] p-4">
          <Story />
        </div>
      </StoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Closed: Story = {
  render: () => (
    <div className="space-y-3">
      <p className="text-sm text-text-muted">
        Press <kbd className="rounded border border-surface-border px-1.5 py-0.5">⌘K</kbd> /{' '}
        <kbd className="rounded border border-surface-border px-1.5 py-0.5">Ctrl+K</kbd> to open.
      </p>
      <CommandPalette
        groups={navGroups}
        open={false}
        onOpenChange={() => undefined}
        onOpenSettings={() => undefined}
        onOpenHelp={() => undefined}
        onToggleTheme={() => undefined}
        isDark={true}
      />
    </div>
  ),
};

export const Open: Story = {
  render: () => (
    <CommandPalette
      groups={navGroups}
      open={true}
      onOpenChange={() => undefined}
      onOpenSettings={() => undefined}
      onOpenHelp={() => undefined}
      onToggleTheme={() => undefined}
      isDark={true}
    />
  ),
};

export const Interactive: Story = {
  render: () => {
    const [open, setOpen] = useState(false);
    return (
      <div className="space-y-3">
        <Button onClick={() => setOpen(true)}>Open command palette</Button>
        <CommandPalette
          groups={navGroups}
          open={open}
          onOpenChange={setOpen}
          onOpenSettings={() => undefined}
          onOpenHelp={() => undefined}
          onToggleTheme={() => undefined}
          isDark={true}
        />
      </div>
    );
  },
};

export const WithExtraActions: Story = {
  render: () => (
    <CommandPalette
      groups={navGroups}
      open={true}
      onOpenChange={() => undefined}
      onOpenSettings={() => undefined}
      onOpenHelp={() => undefined}
      onToggleTheme={() => undefined}
      isDark={false}
      extraActions={[
        {
          id: 'run-rfc2544',
          label: 'Run RFC 2544 benchmark',
          hint: 'modules',
          perform: () => undefined,
        },
        {
          id: 'export-report',
          label: 'Export report',
          hint: 'reports',
          perform: () => undefined,
        },
      ]}
    />
  ),
};
