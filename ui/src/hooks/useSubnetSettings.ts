/**
 * useSubnetSettings
 *
 * Manages the list of configured network-discovery subnets on the
 * /api/v1/security/devices/subnets endpoint. Owns the subnet list state,
 * the new-subnet form fields, the save-status, and the
 * fetch/add/toggle/delete callbacks. Previously inline in
 * SettingsDrawer.
 */

import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { api } from '../api';
import { LogComponents, logger } from '../lib/logger';
import type { SaveStatus, SubnetConfig } from '../types/settings';

const API_BASE: string = import.meta.env.VITE_API_BASE || '';

interface UseSubnetSettingsResult {
  subnets: SubnetConfig[];
  newSubnetCidr: string;
  setNewSubnetCidr: (value: string) => void;
  newSubnetName: string;
  setNewSubnetName: (value: string) => void;
  subnetError: string | null;
  setSubnetError: (value: string | null) => void;
  subnetsStatus: SaveStatus;
  fetchSubnets: () => Promise<void>;
  addSubnet: () => Promise<void>;
  toggleSubnet: (cidr: string, enabled: boolean) => Promise<void>;
  deleteSubnet: (cidr: string) => Promise<void>;
}

export function useSubnetSettings(): UseSubnetSettingsResult {
  const { t } = useTranslation('settings');
  const [subnets, setSubnets] = useState<SubnetConfig[]>([]);
  const [newSubnetCidr, setNewSubnetCidr] = useState('');
  const [newSubnetName, setNewSubnetName] = useState('');
  const [subnetError, setSubnetError] = useState<string | null>(null);
  const [subnetsStatus, setSubnetsStatus] = useState<SaveStatus>('idle');

  const fetchSubnets = async (): Promise<void> => {
    await fetch(`${API_BASE}/api/v1/security/devices/subnets`, { credentials: 'include' })
      .then(async (response) => {
        if (response.ok) {
          const data = await (response.json() as Promise<Record<string, unknown>>);
          setSubnets(Array.isArray(data) ? data : []);
        }
      })
      .catch((err: unknown) => {
        logger.error(LogComponents.DISCOVERY, 'Failed to fetch subnets', err);
      });
  };

  const addSubnet = async (): Promise<void> => {
    const cidr = newSubnetCidr.trim();
    if (!cidr) {
      setSubnetError(t('network.cidrRequired'));
      return;
    }

    setSubnetError(null);
    setSubnetsStatus('saving');

    await api
      .post('/api/v1/security/devices/subnets', {
        cidr,
        name: newSubnetName.trim() || cidr,
        enabled: true,
      })
      .then(async () => {
        setNewSubnetCidr('');
        setNewSubnetName('');
        setSubnetsStatus('saved');
        setTimeout(() => setSubnetsStatus('idle'), 2000);
        await fetchSubnets();
      })
      .catch((err: unknown) => {
        // The client carries the server's own message through, which is what the
        // hand-rolled branch above used to dig out of the body.
        setSubnetError(err instanceof Error ? err.message : 'Failed to add subnet');
        setSubnetsStatus('error');
      });
  };

  const toggleSubnet = async (cidr: string, enabled: boolean): Promise<void> => {
    setSubnetsStatus('saving');
    try {
      await api.put('/api/v1/security/devices/subnets', { cidr, enabled });

      setSubnetsStatus('saved');
      setTimeout(() => setSubnetsStatus('idle'), 2000);
      await fetchSubnets();
    } catch {
      setSubnetsStatus('error');
    }
  };

  const deleteSubnet = async (cidr: string): Promise<void> => {
    setSubnetsStatus('saving');
    try {
      // Backend expects CIDR as query parameter, not in body
      await api.delete(`/api/v1/security/devices/subnets?cidr=${encodeURIComponent(cidr)}`);

      setSubnetsStatus('saved');
      setTimeout(() => setSubnetsStatus('idle'), 2000);
      await fetchSubnets();
    } catch {
      setSubnetsStatus('error');
    }
  };

  return {
    subnets,
    newSubnetCidr,
    setNewSubnetCidr,
    newSubnetName,
    setNewSubnetName,
    subnetError,
    setSubnetError,
    subnetsStatus,
    fetchSubnets,
    addSubnet,
    toggleSubnet,
    deleteSubnet,
  };
}
