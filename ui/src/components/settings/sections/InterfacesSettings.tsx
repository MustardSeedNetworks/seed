/**
 * InterfacesSettings Component
 *
 * Settings → Network Interfaces panel for the multi_interface feature
 * (seed#1192). Lists the active profile's ethernet + Wi-Fi interfaces
 * and lets the operator add, remove, and set-active. Free / Starter
 * tiers are capped server-side at 1 ethernet + 1 Wi-Fi; the add button
 * is disabled with an explanatory message once the cap is reached and
 * the operator lacks multi_interface.
 *
 * The actual probe-loop fan-out across multiple interfaces is a
 * follow-up on the monitor pool — this panel persists the slice; the
 * primary (active) interface continues to be the one that's actively
 * monitored today.
 */

import type React from 'react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useLicense } from '../../../contexts/LicenseContext';
import { useProfileContext } from '../../../contexts/profileContext';
import { Button } from '../../ui/Button';
import { CollapsibleSection } from '../../ui/CollapsibleSection';
import { Input } from '../../ui/Input';
import { Check, Plus, Trash2, Wifi } from '../../ui/icons';

type IfaceKind = 'ethernet' | 'wifi';

export function InterfacesSettings(): React.ReactElement {
  const { t } = useTranslation(['settings', 'errors']);
  const { status: licenseStatus } = useLicense();
  const {
    getAllEthernetInterfaces,
    getAllWifiInterfaces,
    getEthernetInterface,
    getWifiInterface,
    addEthernetInterface,
    addWifiInterface,
    removeEthernetInterface,
    removeWifiInterface,
    setActiveEthernetInterface,
    setActiveWifiInterface,
  } = useProfileContext();

  const [newEthernetName, setNewEthernetName] = useState('');
  const [newWifiName, setNewWifiName] = useState('');
  const [error, setError] = useState<string | null>(null);

  const ethernetInterfaces = getAllEthernetInterfaces();
  const wifiInterfaces = getAllWifiInterfaces();
  const activeEthernet = getEthernetInterface();
  const activeWifi = getWifiInterface();

  const hasMultiInterface = Boolean(licenseStatus?.features?.includes?.('multi_interface'));

  // Operator may always have 1 ethernet + 1 wifi without multi_interface;
  // adding a SECOND of either type is the gated action.
  const canAddEthernet = hasMultiInterface || ethernetInterfaces.length < 1;
  const canAddWifi = hasMultiInterface || wifiInterfaces.length < 1;

  const handleAdd = useCallback(
    async (kind: IfaceKind, name: string): Promise<void> => {
      const trimmed = name.trim();
      if (!trimmed) return;
      setError(null);
      const ok =
        kind === 'ethernet'
          ? await addEthernetInterface(trimmed, true)
          : await addWifiInterface(trimmed, true);
      if (!ok) {
        setError(t('errors:profile.multiInterfaceRequired'));
        return;
      }
      if (kind === 'ethernet') setNewEthernetName('');
      else setNewWifiName('');
    },
    [addEthernetInterface, addWifiInterface, t],
  );

  const handleRemove = useCallback(
    async (kind: IfaceKind, name: string): Promise<void> => {
      const ok = window.confirm(
        t('settings:interfaces.removeConfirm', { name }) +
          '\n\n' +
          t('settings:interfaces.removeConfirmPrompt'),
      );
      if (!ok) return;
      setError(null);
      if (kind === 'ethernet') {
        await removeEthernetInterface(name);
      } else {
        await removeWifiInterface(name);
      }
    },
    [removeEthernetInterface, removeWifiInterface, t],
  );

  const handleSetActive = useCallback(
    async (kind: IfaceKind, name: string): Promise<void> => {
      setError(null);
      if (kind === 'ethernet') {
        await setActiveEthernetInterface(name);
      } else {
        await setActiveWifiInterface(name);
      }
    },
    [setActiveEthernetInterface, setActiveWifiInterface],
  );

  return (
    <CollapsibleSection
      title={
        <div className="inline-flex items-center gap-2">
          <Wifi className="w-4 h-4" />
          <span>{t('settings:interfaces.title')}</span>
        </div>
      }
      defaultOpen={false}
    >
      <div className="stack-sm" data-testid="interfaces-settings-section">
        <p className="text-sm text-text-secondary">{t('settings:interfaces.description')}</p>

        {(!canAddEthernet || !canAddWifi) && (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm text-amber-200">
            {t('settings:interfaces.limitReachedFreeStarter')}
          </div>
        )}

        {error && (
          <div
            className="rounded-lg border border-status-error/30 bg-status-error/5 p-3 text-sm text-status-error"
            data-testid="interfaces-error"
          >
            {error}
          </div>
        )}

        <InterfaceGroup
          kind="ethernet"
          label={t('settings:interfaces.ethernet')}
          interfaces={ethernetInterfaces}
          activeName={activeEthernet?.name ?? ''}
          newName={newEthernetName}
          setNewName={setNewEthernetName}
          canAdd={canAddEthernet}
          onAdd={() => void handleAdd('ethernet', newEthernetName)}
          onRemove={(name) => void handleRemove('ethernet', name)}
          onSetActive={(name) => void handleSetActive('ethernet', name)}
        />

        <InterfaceGroup
          kind="wifi"
          label={t('settings:interfaces.wifi')}
          interfaces={wifiInterfaces}
          activeName={activeWifi?.name ?? ''}
          newName={newWifiName}
          setNewName={setNewWifiName}
          canAdd={canAddWifi}
          onAdd={() => void handleAdd('wifi', newWifiName)}
          onRemove={(name) => void handleRemove('wifi', name)}
          onSetActive={(name) => void handleSetActive('wifi', name)}
        />
      </div>
    </CollapsibleSection>
  );
}

interface InterfaceGroupProps {
  kind: IfaceKind;
  label: string;
  interfaces: { name: string; enabled: boolean }[];
  activeName: string;
  newName: string;
  setNewName: (s: string) => void;
  canAdd: boolean;
  onAdd: () => void;
  onRemove: (name: string) => void;
  onSetActive: (name: string) => void;
}

function InterfaceGroup({
  kind,
  label,
  interfaces,
  activeName,
  newName,
  setNewName,
  canAdd,
  onAdd,
  onRemove,
  onSetActive,
}: InterfaceGroupProps): React.ReactElement {
  const { t } = useTranslation('settings');
  const placeholder = kind === 'ethernet' ? 'eth0' : 'wlan0';

  return (
    <div className="stack-xs" data-testid={`interface-group-${kind}`}>
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium">{label}</h4>
        <span className="text-xs text-text-muted">{interfaces.length}</span>
      </div>

      {interfaces.length === 0 ? (
        <div className="text-sm text-text-muted italic">
          {t('interfaces.noInterfaces', { kind: label })}
        </div>
      ) : (
        <ul className="stack-xs">
          {interfaces.map((iface) => {
            const isActive = iface.name === activeName;
            return (
              <li
                key={iface.name}
                className="flex items-center gap-2 rounded border border-surface-border p-2"
                data-testid={`interface-row-${kind}-${iface.name}`}
              >
                <span className="font-mono text-sm flex-1">{iface.name}</span>
                {isActive && (
                  <span className="rounded bg-status-success/10 px-1.5 py-0.5 text-xs text-status-success">
                    {t('interfaces.active')}
                  </span>
                )}
                {!isActive && (
                  <Button
                    variant="ghost"
                    tone="gray"
                    size="xs"
                    leftIcon={<Check className="w-3 h-3" />}
                    onClick={() => onSetActive(iface.name)}
                    data-testid={`set-active-${kind}-${iface.name}`}
                  >
                    {t('interfaces.setActive')}
                  </Button>
                )}
                <Button
                  variant="ghost"
                  tone="red"
                  size="xs"
                  leftIcon={<Trash2 className="w-3 h-3" />}
                  onClick={() => onRemove(iface.name)}
                  data-testid={`remove-${kind}-${iface.name}`}
                >
                  {t('interfaces.removeInterface')}
                </Button>
              </li>
            );
          })}
        </ul>
      )}

      <div className="flex items-end gap-2">
        <div className="flex-1">
          <Input
            id={`add-${kind}-name`}
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder={placeholder}
            maxLength={32}
            disabled={!canAdd}
            data-testid={`add-${kind}-input`}
          />
        </div>
        <Button
          variant="solid"
          tone="violet"
          size="sm"
          leftIcon={<Plus className="w-3 h-3" />}
          disabled={!canAdd || !newName.trim()}
          onClick={onAdd}
          data-testid={`add-${kind}-button`}
          title={canAdd ? undefined : t('interfaces.limitReachedFreeStarter')}
        >
          {kind === 'ethernet' ? t('interfaces.addEthernet') : t('interfaces.addWifi')}
        </Button>
      </div>
    </div>
  );
}
