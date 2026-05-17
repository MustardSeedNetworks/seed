/**
 * useSubnetSettings
 *
 * Manages the list of configured network-discovery subnets on the
 * /api/v1/shell/devices/subnets endpoint. Owns the subnet list state,
 * the new-subnet form fields, the save-status, and the
 * fetch/add/toggle/delete callbacks. Previously inline in
 * SettingsDrawer.
 */

import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
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

export function useSubnetSettings(isOpen: boolean): UseSubnetSettingsResult {
  const { t } = useTranslation('settings');
  const [subnets, setSubnets] = useState<SubnetConfig[]>([]);
  const [newSubnetCidr, setNewSubnetCidr] = useState('');
  const [newSubnetName, setNewSubnetName] = useState('');
  const [subnetError, setSubnetError] = useState<string | null>(null);
  const [subnetsStatus, setSubnetsStatus] = useState<SaveStatus>('idle');

  const fetchSubnets = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/subnets`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await (response.json() as Promise<Record<string, unknown>>);
        setSubnets(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      logger.error(LogComponents.Discovery, 'Failed to fetch subnets', err);
    }
  }, []);

  // Fetch subnets when drawer opens
  useEffect(() => {
    if (isOpen) {
      fetchSubnets().catch(() => undefined);
    }
  }, [isOpen, fetchSubnets]);

  const addSubnet = useCallback(async (): Promise<void> => {
    if (!newSubnetCidr.trim()) {
      setSubnetError(t('network.cidrRequired'));
      return;
    }

    setSubnetError(null);
    setSubnetsStatus('saving');

    try {
      const response = await fetch(`${API_BASE}/api/v1/shell/devices/subnets`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          cidr: newSubnetCidr.trim(),
          name: newSubnetName.trim() || newSubnetCidr.trim(),
          enabled: true,
        }),
      });

      if (response.ok) {
        setNewSubnetCidr('');
        setNewSubnetName('');
        setSubnetsStatus('saved');
        setTimeout(() => setSubnetsStatus('idle'), 2000);
        await fetchSubnets();
      } else {
        // Handle both JSON and plain text error responses
        const contentType = response.headers.get('content-type');
        if (contentType?.includes('application/json')) {
          const errorData = await (response.json() as Promise<{ error?: string }>);
          setSubnetError(errorData.error || 'Failed to add subnet');
        } else {
          const errorText = await (response.text() as Promise<string>);
          setSubnetError(errorText || 'Failed to add subnet');
        }
        setSubnetsStatus('error');
      }
    } catch (err) {
      setSubnetError(err instanceof Error ? err.message : 'Network error adding subnet');
      setSubnetsStatus('error');
    }
  }, [newSubnetCidr, newSubnetName, fetchSubnets, t]);

  const toggleSubnet = useCallback(
    async (cidr: string, enabled: boolean): Promise<void> => {
      setSubnetsStatus('saving');
      try {
        const response = await fetch(`${API_BASE}/api/v1/shell/devices/subnets`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ cidr, enabled }),
        });

        if (response.ok) {
          setSubnetsStatus('saved');
          setTimeout(() => setSubnetsStatus('idle'), 2000);
          await fetchSubnets();
        } else {
          setSubnetsStatus('error');
        }
      } catch {
        setSubnetsStatus('error');
      }
    },
    [fetchSubnets],
  );

  const deleteSubnet = useCallback(
    async (cidr: string): Promise<void> => {
      setSubnetsStatus('saving');
      try {
        // Backend expects CIDR as query parameter, not in body
        const response = await fetch(
          `${API_BASE}/api/v1/shell/devices/subnets?cidr=${encodeURIComponent(cidr)}`,
          {
            method: 'DELETE',
            credentials: 'include',
          },
        );

        if (response.ok) {
          setSubnetsStatus('saved');
          setTimeout(() => setSubnetsStatus('idle'), 2000);
          await fetchSubnets();
        } else {
          setSubnetsStatus('error');
        }
      } catch {
        setSubnetsStatus('error');
      }
    },
    [fetchSubnets],
  );

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
