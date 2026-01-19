/**
 * Slider Component - Example Usage
 *
 * This file demonstrates various ways to use the Slider component
 * in real-world scenarios within the Seed project.
 */

import type React from 'react';
import { useState } from 'react';
import { Slider } from './Slider';

/**
 * Example 1: Scanner Configuration Panel
 * Shows how to use multiple sliders together for complex settings
 */
export function ScannerConfigExample(): React.JSX.Element {
  const [probeInterval, setProbeInterval] = useState(75);
  const [scanTimeout, setScanTimeout] = useState(2000);
  const [workers, setWorkers] = useState(20);
  const [rescanInterval, setRescanInterval] = useState(10);
  const [bannerTimeout, setBannerTimeout] = useState(2000);

  return (
    <div class="space-y-6 max-w-2xl p-6 bg-surface-raised rounded-lg border border-surface-border">
      <h3 class="heading-3">Network Scanner Settings</h3>

      {/* Probe Interval: 25ms - 500ms */}
      <Slider
        value={probeInterval}
        onChange={setProbeInterval}
        min={25}
        max={500}
        step={25}
        label="Probe Interval"
        leftLabel="Faster"
        rightLabel="Slower"
        formatValue={(v: number): string => `${v}ms`}
      />

      {/* Scan Timeout: 500ms - 10s */}
      <Slider
        value={scanTimeout}
        onChange={setScanTimeout}
        min={500}
        max={10000}
        step={500}
        label="Scan Timeout"
        leftLabel="Quick"
        rightLabel="Patient"
        formatValue={(v: number): string => (v >= 1000 ? `${v / 1000}s` : `${v}ms`)}
      />

      {/* Workers: 5 - 100 */}
      <Slider
        value={workers}
        onChange={setWorkers}
        min={5}
        max={100}
        step={5}
        label="Worker Threads"
        leftLabel="Conservative"
        rightLabel="Aggressive"
        formatValue={(v: number): string => `${v} workers`}
      />

      {/* Rescan Interval: 1min - 60min */}
      <Slider
        value={rescanInterval}
        onChange={setRescanInterval}
        min={1}
        max={60}
        step={1}
        label="Rescan Interval"
        leftLabel="Frequent"
        rightLabel="Rare"
        formatValue={(v: number): string => `${v} min`}
      />

      {/* Banner Timeout: 500ms - 10s */}
      <Slider
        value={bannerTimeout}
        onChange={setBannerTimeout}
        min={500}
        max={10000}
        step={500}
        label="Banner Timeout"
        leftLabel="Quick"
        rightLabel="Patient"
        formatValue={(v: number): string => `${v}ms`}
      />

      {/* Configuration Summary */}
      <div class="mt-8 p-4 bg-surface-base rounded border border-surface-border">
        <h4 class="heading-4 mb-2">Current Configuration</h4>
        <div class="body-small space-y-1 font-mono">
          <div>
            <span class="text-text-muted">Probe Interval:</span> {probeInterval}ms
          </div>
          <div>
            <span class="text-text-muted">Scan Timeout:</span> {scanTimeout}
            ms
          </div>
          <div>
            <span class="text-text-muted">Workers:</span> {workers}
          </div>
          <div>
            <span class="text-text-muted">Rescan Interval:</span> {rescanInterval} min
          </div>
          <div>
            <span class="text-text-muted">Banner Timeout:</span> {bannerTimeout}ms
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * Example 2: Simple Timeout Setting
 * Basic usage with millisecond formatting
 */
export function TimeoutSettingExample(): React.JSX.Element {
  const [timeout, setTimeout] = useState(2000);

  return (
    <div class="max-w-md p-4 bg-surface-raised rounded-lg">
      <Slider
        value={timeout}
        onChange={setTimeout}
        min={500}
        max={10000}
        step={500}
        label="Request Timeout"
        formatValue={(v: number): string => `${v}ms`}
      />
    </div>
  );
}

/**
 * Example 3: Worker Thread Configuration
 * Shows count-based slider with labels
 */
export function WorkerConfigExample(): React.JSX.Element {
  const [workers, setWorkers] = useState(20);

  return (
    <div class="max-w-md p-4 bg-surface-raised rounded-lg">
      <Slider
        value={workers}
        onChange={setWorkers}
        min={5}
        max={100}
        step={5}
        label="Concurrent Workers"
        leftLabel="Conservative"
        rightLabel="Aggressive"
        formatValue={(v: number): string => `${v} workers`}
      />
      <p class="mt-2 caption text-text-muted">
        More workers = faster scanning, but higher CPU usage
      </p>
    </div>
  );
}

/**
 * Example 4: Percentage Slider
 * Common pattern for volume, brightness, etc.
 */
export function PercentageSliderExample(): React.JSX.Element {
  const [volume, setVolume] = useState(75);

  return (
    <div class="max-w-md p-4 bg-surface-raised rounded-lg">
      <Slider
        value={volume}
        onChange={setVolume}
        min={0}
        max={100}
        step={5}
        label="Alert Volume"
        leftLabel="Quiet"
        rightLabel="Loud"
        formatValue={(v: number): string => `${v}%`}
      />
    </div>
  );
}

/**
 * Example 5: Disabled State
 * Shows how to disable a slider based on conditions
 */
export function DisabledSliderExample(): React.JSX.Element {
  const [autoScan, setAutoScan] = useState(false);
  const [interval, setInterval] = useState(10);

  return (
    <div class="max-w-md p-4 bg-surface-raised rounded-lg space-y-4">
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          type="checkbox"
          checked={autoScan}
          onChange={(e: React.ChangeEvent<HTMLInputElement>): void => setAutoScan(e.target.checked)}
          class="w-4 h-4"
        />
        <span class="body">Enable Auto-Scan</span>
      </label>

      <Slider
        value={interval}
        onChange={setInterval}
        min={1}
        max={60}
        step={1}
        label="Auto-Scan Interval"
        leftLabel="Frequent"
        rightLabel="Rare"
        formatValue={(v: number): string => `${v} min`}
        disabled={!autoScan}
      />
    </div>
  );
}

/**
 * Example 6: Time Duration with Smart Formatting
 * Automatically switches between ms/s/min based on value
 */
export function SmartDurationExample(): React.JSX.Element {
  const [duration, setDuration] = useState(30000);

  const formatDuration = (ms: number): string => {
    if (ms < 1000) {
      return `${ms}ms`;
    }
    if (ms < 60000) {
      return `${(ms / 1000).toFixed(1)}s`;
    }
    return `${(ms / 60000).toFixed(1)}min`;
  };

  return (
    <div class="max-w-md p-4 bg-surface-raised rounded-lg">
      <Slider
        value={duration}
        onChange={setDuration}
        min={100}
        max={300000}
        step={100}
        label="Cache Duration"
        formatValue={formatDuration}
      />
    </div>
  );
}

/**
 * Example 7: Integration with Form State
 * Shows how to integrate with form libraries or state management
 */
export function FormIntegrationExample(): React.JSX.Element {
  const [config, setConfig] = useState({
    timeout: 2000,
    retries: 3,
    workers: 20,
  });

  const updateConfig = (key: keyof typeof config, value: number): void => {
    setConfig((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent): void => {
    e.preventDefault();
    // Here you would send config to backend or save to settings
  };

  return (
    <form
      onSubmit={handleSubmit}
      class="max-w-md p-6 bg-surface-raised rounded-lg border border-surface-border space-y-6"
    >
      <h3 class="heading-3">Network Settings</h3>

      <Slider
        value={config.timeout}
        onChange={(v: number): void => updateConfig('timeout', v)}
        min={500}
        max={10000}
        step={500}
        label="Request Timeout"
        formatValue={(v: number): string => `${v}ms`}
      />

      <Slider
        value={config.retries}
        onChange={(v: number): void => updateConfig('retries', v)}
        min={0}
        max={10}
        step={1}
        label="Max Retries"
        formatValue={(v: number): string => `${v} attempts`}
      />

      <Slider
        value={config.workers}
        onChange={(v: number): void => updateConfig('workers', v)}
        min={5}
        max={100}
        step={5}
        label="Worker Threads"
        formatValue={(v: number): string => `${v} workers`}
      />

      <button
        type="submit"
        class="w-full px-4 py-2 bg-brand-primary text-text-inverse rounded-md hover:bg-brand-accent transition-colors"
      >
        Save Settings
      </button>
    </form>
  );
}
