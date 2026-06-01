/**
 * AlertRulesPage
 *
 * Operator-facing CRUD over /api/v1/alert-rules (#1384). Rules
 * defined here drive the listener alert pipeline at runtime — see
 * internal/alerts/pipeline/listener_rules_loader.go for the
 * DB-only-when-non-empty semantics.
 *
 * Template syntax in AlertTitle / AlertMessage is documented inline:
 *   {{.SourceAddr}} {{.Severity}} {{.Kind}} {{.MatchedRuleName}}
 *   {{.Payload.<field>}} from the event's JSON payload
 */

import { AlertTriangle, Plus, X } from 'lucide-react';
import { type FormEvent, type JSX, useState } from 'react';
import { useAlertRules } from '../hooks/useAlertRules';
import type { AlertRule, AlertRuleInput } from '../types/alertRules';
import { Breadcrumbs } from '../ui/Breadcrumbs';
import { PageHeader } from '../ui/PageHeader';

const MATCH_KINDS = ['', 'syslog-udp', 'snmp-trap-v2c'] as const;
const ALERT_TYPES = ['system', 'connectivity', 'security', 'performance'] as const;
const SEVERITIES = ['info', 'warning', 'error', 'critical'] as const;

export function AlertRulesPage(): JSX.Element {
  const { rules, loading, error, create, update, remove } = useAlertRules();
  const [editing, setEditing] = useState<AlertRule | null>(null);
  const [showCreate, setShowCreate] = useState<boolean>(false);

  return (
    <section className="stack-xl">
      <Breadcrumbs />
      <PageHeader
        icon={AlertTriangle}
        title="Alert rules"
        description="Operator-defined rules drive the listener alert pipeline. When this list is empty the pipeline falls back to built-in defaults."
        iconColorClass="text-amber-400"
      />

      {error ? (
        <div className="rounded-md border border-rose-500/40 bg-rose-500/10 p-3 text-sm text-rose-200">
          {error}
        </div>
      ) : null}

      <div className="flex items-center justify-between">
        <p className="text-sm text-zinc-400">
          {loading ? 'Loading…' : `${rules.length} rule${rules.length === 1 ? '' : 's'}`}
        </p>
        <button
          type="button"
          onClick={(): void => setShowCreate(true)}
          data-testid="alert-rules-add"
          className="inline-flex items-center gap-2 rounded-md bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-500"
        >
          <Plus className="h-4 w-4" />
          Add rule
        </button>
      </div>

      {!loading && rules.length === 0 ? (
        <p className="rounded-md border border-zinc-700 bg-zinc-900 p-4 text-sm text-zinc-400">
          No operator-defined rules yet. The listener pipeline is using the built-in default ruleset
          (severe syslog, linkDown traps, authentication failures). Adding any enabled rule below
          replaces the defaults entirely.
        </p>
      ) : null}

      <ul className="stack-sm" data-testid="alert-rules-list">
        {rules.map((rule) => (
          <li
            key={rule.id}
            data-testid={`alert-rule-row-${rule.id}`}
            className="rounded-md border border-zinc-700 bg-zinc-900 p-4"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="stack-xs">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-zinc-100">{rule.name}</span>
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs ${
                      rule.enabled
                        ? 'bg-emerald-500/20 text-emerald-300'
                        : 'bg-zinc-700 text-zinc-400'
                    }`}
                  >
                    {rule.enabled ? 'enabled' : 'disabled'}
                  </span>
                  <span className="text-xs text-zinc-500">
                    {rule.alertType} / {rule.alertSeverity}
                  </span>
                </div>
                <span className="text-xs text-zinc-400">{rule.alertTitle}</span>
                {rule.windowSeconds > 0 ? (
                  <span className="text-xs text-zinc-500">
                    Fires after {rule.thresholdCount} matches in {rule.windowSeconds}s
                  </span>
                ) : null}
              </div>
              <div className="flex shrink-0 gap-2">
                <button
                  type="button"
                  onClick={(): void => setEditing(rule)}
                  className="rounded-md border border-zinc-600 px-2 py-1 text-xs text-zinc-200 hover:border-zinc-500"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={(): void => {
                    if (window.confirm(`Delete rule "${rule.name}"?`)) {
                      void remove(rule.id);
                    }
                  }}
                  className="rounded-md border border-rose-500/50 px-2 py-1 text-xs text-rose-300 hover:border-rose-500"
                >
                  Delete
                </button>
              </div>
            </div>
          </li>
        ))}
      </ul>

      {showCreate ? (
        <RuleForm
          mode="create"
          onClose={(): void => setShowCreate(false)}
          onSubmit={async (input): Promise<void> => {
            await create(input);
            setShowCreate(false);
          }}
        />
      ) : null}

      {editing ? (
        <RuleForm
          mode="edit"
          initial={editing}
          onClose={(): void => setEditing(null)}
          onSubmit={async (input): Promise<void> => {
            await update(editing.id, input);
            setEditing(null);
          }}
        />
      ) : null}
    </section>
  );
}

interface RuleFormProps {
  mode: 'create' | 'edit';
  initial?: AlertRule;
  onClose: () => void;
  onSubmit: (input: AlertRuleInput) => Promise<void>;
}

function RuleForm({ mode, initial, onClose, onSubmit }: RuleFormProps): JSX.Element {
  const [name, setName] = useState<string>(initial?.name ?? '');
  const [enabled, setEnabled] = useState<boolean>(initial?.enabled ?? true);
  const [matchKind, setMatchKind] = useState<string>(initial?.matchKind ?? '');
  const [matchSeverity, setMatchSeverity] = useState<string>(initial?.matchSeverity ?? '');
  const [matchPayloadContains, setMatchPayloadContains] = useState<string>(
    initial?.matchPayloadContains ?? '',
  );
  const [alertType, setAlertType] = useState<string>(initial?.alertType ?? 'system');
  const [alertSeverity, setAlertSeverity] = useState<string>(initial?.alertSeverity ?? 'error');
  const [alertTitle, setAlertTitle] = useState<string>(initial?.alertTitle ?? '');
  const [alertMessage, setAlertMessage] = useState<string>(initial?.alertMessage ?? '');
  const [windowSeconds, setWindowSeconds] = useState<number>(initial?.windowSeconds ?? 0);
  const [thresholdCount, setThresholdCount] = useState<number>(initial?.thresholdCount ?? 1);
  const [submitting, setSubmitting] = useState<boolean>(false);
  const [formError, setFormError] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>): Promise<void> => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit({
        name,
        enabled,
        matchKind,
        matchSeverity,
        matchPayloadContains,
        alertType,
        alertSeverity,
        alertTitle,
        alertMessage,
        windowSeconds,
        thresholdCount,
      });
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save rule');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-zinc-950/70 p-4">
      <form
        onSubmit={handleSubmit}
        data-testid="alert-rule-form"
        className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg border border-zinc-700 bg-zinc-900 p-6 stack-md"
      >
        <div className="flex items-start justify-between">
          <h2 className="text-lg font-semibold text-zinc-100">
            {mode === 'create' ? 'Add alert rule' : `Edit rule: ${initial?.name ?? ''}`}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="text-zinc-400 hover:text-zinc-200"
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {formError ? (
          <div className="rounded-md border border-rose-500/40 bg-rose-500/10 p-2 text-sm text-rose-200">
            {formError}
          </div>
        ) : null}

        <Field label="Name" required>
          <input
            type="text"
            required
            value={name}
            onChange={(e): void => setName(e.target.value)}
            className={inputClass}
          />
        </Field>

        <label className="flex items-center gap-2 text-sm text-zinc-200">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e): void => setEnabled(e.target.checked)}
          />
          Enabled
        </label>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Match kind">
            <select
              value={matchKind}
              onChange={(e): void => setMatchKind(e.target.value)}
              className={inputClass}
            >
              {MATCH_KINDS.map((k) => (
                <option key={k} value={k}>
                  {k === '' ? '(any)' : k}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Match severity">
            <input
              type="text"
              placeholder="(any)"
              value={matchSeverity}
              onChange={(e): void => setMatchSeverity(e.target.value)}
              className={inputClass}
            />
          </Field>
        </div>

        <Field label="Match payload contains (substring)">
          <input
            type="text"
            placeholder="(any)"
            value={matchPayloadContains}
            onChange={(e): void => setMatchPayloadContains(e.target.value)}
            className={inputClass}
          />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Alert type" required>
            <select
              value={alertType}
              onChange={(e): void => setAlertType(e.target.value)}
              className={inputClass}
            >
              {ALERT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Alert severity" required>
            <select
              value={alertSeverity}
              onChange={(e): void => setAlertSeverity(e.target.value)}
              className={inputClass}
            >
              {SEVERITIES.map((s) => (
                <option key={s} value={s}>
                  {s}
                </option>
              ))}
            </select>
          </Field>
        </div>

        <Field
          label="Alert title (supports {{.SourceAddr}}, {{.Severity}}, {{.Payload.<field>}})"
          required
        >
          <input
            type="text"
            required
            value={alertTitle}
            onChange={(e): void => setAlertTitle(e.target.value)}
            className={inputClass}
          />
        </Field>

        <Field label="Alert message (template syntax also supported)">
          <textarea
            rows={2}
            value={alertMessage}
            onChange={(e): void => setAlertMessage(e.target.value)}
            className={inputClass}
          />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Window seconds (0 = fire on first match)">
            <input
              type="number"
              min={0}
              value={windowSeconds}
              onChange={(e): void => setWindowSeconds(Number(e.target.value))}
              className={inputClass}
            />
          </Field>
          <Field label="Threshold count (1 = single match)">
            <input
              type="number"
              min={1}
              value={thresholdCount}
              onChange={(e): void => setThresholdCount(Number(e.target.value))}
              className={inputClass}
            />
          </Field>
        </div>

        <div className="flex justify-end gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-zinc-600 px-3 py-2 text-sm text-zinc-200"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-emerald-600 px-3 py-2 text-sm font-medium text-white hover:bg-emerald-500 disabled:opacity-50"
          >
            {submitting ? 'Saving…' : mode === 'create' ? 'Create' : 'Save'}
          </button>
        </div>
      </form>
    </div>
  );
}

const inputClass =
  'w-full rounded-md border border-zinc-600 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-500';

interface FieldProps {
  label: string;
  required?: boolean;
  children: JSX.Element;
}

function Field({ label, required, children }: FieldProps): JSX.Element {
  return (
    <div className="flex flex-col gap-1 text-sm text-zinc-300">
      <span>
        {label}
        {required ? <span className="ml-1 text-rose-400">*</span> : null}
      </span>
      {children}
    </div>
  );
}
