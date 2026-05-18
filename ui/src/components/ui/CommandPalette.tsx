/**
 * CommandPalette — keyboard-first command/navigation palette (cmdk).
 *
 * Mount at the AppShell level. Opens on Cmd+K (macOS) / Ctrl+K (others)
 * or when programmatically opened by external state. Populates with:
 *   - All sidebar nav entries (jump to page)
 *   - Common actions (Open Settings, Open Help, Toggle Theme)
 *
 * Tokens resolve via the same `text-text-primary` / `bg-bg-surface`
 * aliases shared by seed/stem/niac.
 */
import { Command } from 'cmdk';
import { HelpCircle, Moon, Search, Settings as SettingsIcon, Sun } from 'lucide-react';
import { type FC, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { SidebarNavGroup } from '../../ui/Sidebar';

export interface CommandPaletteAction {
  id: string;
  label: string;
  hint?: string;
  icon?: typeof SettingsIcon;
  perform: () => void;
}

export interface CommandPaletteProps {
  /** Sidebar groups — same shape as Sidebar consumes. */
  groups: SidebarNavGroup[];
  /** Whether the palette is open. Controlled externally so AppShell can
   *  also open it from a header button. */
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Optional extra actions appended below the auto-generated ones. */
  extraActions?: CommandPaletteAction[];
  /** Callbacks for the built-in actions. Pass `undefined` to hide. */
  onOpenSettings?: () => void;
  onOpenHelp?: () => void;
  onToggleTheme?: () => void;
  /** Theme state — used to label the "Toggle Theme" action. */
  isDark?: boolean;
}

export const CommandPalette: FC<CommandPaletteProps> = ({
  groups,
  open,
  onOpenChange,
  extraActions = [],
  onOpenSettings,
  onOpenHelp,
  onToggleTheme,
  isDark,
}) => {
  const navigate = useNavigate();
  const [value, setValue] = useState('');

  // Global ⌘K / Ctrl+K keybinding.
  useEffect(() => {
    const handler = (e: KeyboardEvent): void => {
      if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
        e.preventDefault();
        onOpenChange(!open);
      }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open, onOpenChange]);

  // Reset query when the dialog closes.
  useEffect(() => {
    if (!open) {
      setValue('');
    }
  }, [open]);

  const handleSelect = (perform: () => void): void => {
    perform();
    onOpenChange(false);
  };

  return (
    <Command.Dialog
      open={open}
      onOpenChange={onOpenChange}
      label="Command palette"
      class="fixed inset-0 z-[60] flex items-start justify-center pt-[10vh]"
      shouldFilter={true}
    >
      <button
        type="button"
        class="absolute inset-0 bg-black/70 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
        aria-label="Close command palette"
      />
      <div class="relative mx-4 w-full max-w-xl rounded-2xl border border-surface-border bg-bg-surface/95 shadow-2xl">
        <div class="flex items-center gap-2 border-b border-surface-border px-4 py-3">
          <Search class="h-4 w-4 text-text-muted" aria-hidden="true" />
          <Command.Input
            autoFocus={true}
            value={value}
            onValueChange={setValue}
            placeholder="Search pages and actions…"
            class="flex-1 bg-transparent text-sm text-text-primary placeholder:text-text-muted focus:outline-none"
          />
          <kbd class="hidden sm:inline-flex items-center rounded border border-surface-border px-1.5 py-0.5 text-[11px] text-text-muted">
            esc
          </kbd>
        </div>
        <Command.List class="max-h-[60vh] overflow-y-auto px-2 py-2 text-sm">
          <Command.Empty class="px-3 py-6 text-center text-text-muted">No matches.</Command.Empty>

          {groups.map((group) => (
            <Command.Group
              key={group.label}
              heading={group.label}
              class="px-1 py-1 text-xs uppercase tracking-wider text-text-muted"
            >
              {group.items.map((item) => {
                const Icon = item.icon;
                return (
                  <Command.Item
                    key={item.path}
                    value={`${group.label} ${item.label} ${item.path}`}
                    onSelect={() =>
                      handleSelect(() => {
                        const result = navigate(item.path) as void | Promise<void>;
                        if (result instanceof Promise) {
                          result.catch(() => undefined);
                        }
                      })
                    }
                    class="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-text-primary aria-selected:bg-surface-hover"
                  >
                    <Icon class="h-4 w-4 text-text-muted" aria-hidden="true" />
                    <span class="flex-1">{item.label}</span>
                    <span class="text-xs text-text-muted">{item.path}</span>
                  </Command.Item>
                );
              })}
            </Command.Group>
          ))}

          <Command.Group
            heading="Actions"
            class="px-1 py-1 text-xs uppercase tracking-wider text-text-muted"
          >
            {onOpenSettings ? (
              <Command.Item
                value="open settings"
                onSelect={() => handleSelect(onOpenSettings)}
                class="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-text-primary aria-selected:bg-surface-hover"
              >
                <SettingsIcon class="h-4 w-4 text-text-muted" aria-hidden="true" />
                <span>Open Settings</span>
              </Command.Item>
            ) : null}
            {onOpenHelp ? (
              <Command.Item
                value="open help"
                onSelect={() => handleSelect(onOpenHelp)}
                class="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-text-primary aria-selected:bg-surface-hover"
              >
                <HelpCircle class="h-4 w-4 text-text-muted" aria-hidden="true" />
                <span>Open Help</span>
              </Command.Item>
            ) : null}
            {onToggleTheme ? (
              <Command.Item
                value="toggle theme dark light mode"
                onSelect={() => handleSelect(onToggleTheme)}
                class="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-text-primary aria-selected:bg-surface-hover"
              >
                {isDark ? (
                  <Sun class="h-4 w-4 text-text-muted" aria-hidden="true" />
                ) : (
                  <Moon class="h-4 w-4 text-text-muted" aria-hidden="true" />
                )}
                <span>{isDark ? 'Switch to light mode' : 'Switch to dark mode'}</span>
              </Command.Item>
            ) : null}
            {extraActions.map((action) => {
              const Icon = action.icon;
              return (
                <Command.Item
                  key={action.id}
                  value={`${action.label} ${action.hint ?? ''}`}
                  onSelect={() => handleSelect(action.perform)}
                  class="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-text-primary aria-selected:bg-surface-hover"
                >
                  {Icon ? (
                    <Icon class="h-4 w-4 text-text-muted" aria-hidden="true" />
                  ) : (
                    <span class="h-4 w-4" />
                  )}
                  <span class="flex-1">{action.label}</span>
                  {action.hint ? <span class="text-xs text-text-muted">{action.hint}</span> : null}
                </Command.Item>
              );
            })}
          </Command.Group>
        </Command.List>
      </div>
    </Command.Dialog>
  );
};
