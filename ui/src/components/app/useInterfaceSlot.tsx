/**
 * The rail footer's one product control: which interface Seed reads. Ethernet
 * interfaces are the options, and Wi-Fi is a mode rather than a list (the page
 * set follows it), so it is one more option. It stays offered with no Wi-Fi
 * hardware so the troubleshooting page's no-hardware view is still reachable.
 */

import { EthernetPort, Wifi } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useProfileContext } from '../../contexts/profileContext';
import type { RailProductSlotProps } from '../../shared/RailProductSlot';
import type { InterfaceInfo } from '../../types/generated/categorized-interfaces-response';

export interface InterfaceSlotArgs {
  interfaces: InterfaceInfo[];
  currentInterface: string;
  isWifi: boolean;
  hasWifiInterface: boolean;
  onInterfaceChange: (interfaceName: string) => void;
  switchToInterfaceType: (type: 'ethernet' | 'wifi') => void;
  recommendedEthernet?: string;
}

const WIFI_VALUE = '\u0000wifi';

/**
 * Technical interface name to a name an operator recognises.
 * enp0s1 -> "Ethernet 1", eth0 -> "Ethernet".
 */
function friendlyEthernetName(name: string): string {
  const numMatch = /(\d+)$/.exec(name);
  const num = numMatch?.[1] ? Number.parseInt(numMatch[1], 10) : 0;
  return num > 0 ? `Ethernet ${num + 1}` : 'Ethernet';
}

export function useInterfaceSlot({
  interfaces,
  currentInterface,
  isWifi,
  hasWifiInterface,
  onInterfaceChange,
  switchToInterfaceType,
  recommendedEthernet,
}: InterfaceSlotArgs): RailProductSlotProps {
  const { t } = useTranslation();
  const { setEthernetInterface } = useProfileContext();

  const options = [
    ...interfaces
      .filter((iface) => iface.type !== 'wifi')
      .map((iface) => ({
        value: iface.name,
        label: [
          friendlyEthernetName(iface.name),
          iface.name,
          iface.name === recommendedEthernet ? `(${t('interface.recommended')})` : '',
        ]
          .filter(Boolean)
          .join(' · '),
      })),
    {
      value: WIFI_VALUE,
      label: hasWifiInterface ? t('interface.wifi') : t('interface.wifiNoHardware'),
    },
  ];

  const select = (value: string): void => {
    // The choice is persisted to the active profile (#754), so it survives a
    // reload rather than only holding for this session.
    if (value === WIFI_VALUE) {
      switchToInterfaceType('wifi');
      return;
    }
    switchToInterfaceType('ethernet');
    onInterfaceChange(value);
    void setEthernetInterface(value, true);
  };

  return {
    label: t('labels.interface'),
    value: isWifi ? t('interface.wifi') : friendlyEthernetName(currentInterface),
    icon: isWifi ? <Wifi /> : <EthernetPort />,
    options,
    selected: isWifi ? WIFI_VALUE : currentInterface,
    onSelect: select,
    'data-testid': 'rail-product-slot',
  };
}
