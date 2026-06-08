# State management — what goes where

The Seed UI uses four state mechanisms. Picking the wrong one is the most common
source of re-render bugs, stale data, and "why are there two sources of truth"
confusion. Use this decision order; when two fit, prefer the one higher in the
list.

## Decision order

1. **Is it server data** (fetched from `/api/**`, owned by the backend)?
   → **React Query** (`@tanstack/react-query`). It owns caching, dedup,
   background refetch, and staleness. Never copy server data into Zustand/Context
   "to keep it handy" — that creates a second source of truth that goes stale.
   - Provider: `main.tsx` (`QueryClientProvider` + `lib/queryClient`).
   - Example: `contexts/profileQueries` (profile fetches).
   - Note: some legacy fetch paths (`hooks/useNetworkFetchers` + SSE) predate
     React Query and push into `AppContext`; new server reads should use React
     Query rather than extend that path.

2. **Is it ephemeral global app/UI state** that many unrelated components read or
   drive, but is NOT server-owned and need NOT persist across reloads?
   → **Zustand store** (`stores/*`). Atomic updates, no provider nesting, no
   re-render cascade, trivially unit-testable.
   - `stores/testRunStore` — the run-all-tests orchestration state machine
     (idle/running/partial). Cross-component (FAB, app shell, cards) signalling
     with zero React context. The reference example for "ephemeral global".
   - `stores/profileStore` — replaced a 48-hook profile context; persists only
     the active profile *id*.

3. **Is it a cross-cutting capability** scoped to the authenticated app —
   identity, permissions, or a small derived bundle handed to a subtree?
   → **React Context**. Use for stable, low-frequency values; avoid putting
   high-churn data in Context (every consumer re-renders).
   - `contexts/AppContext` — the per-render dashboard bundle (cards, current
     interface, display settings) assembled by `useAppOrchestration`.
   - `contexts/RoleContext`, `contexts/LicenseContext`, `contexts/profileContext`.

4. **Is it local to one component / subtree** (open/closed, hovered, form draft)?
   → **`useState` / `useReducer`**. Default for anything not shared. Lift only
   when a second component genuinely needs it; reach for Zustand/Context only when
   lifting would mean prop-drilling more than ~2 levels.

## Anti-patterns

- **Duplicating server data** into Zustand/Context. React Query is the cache.
- **Global store for one component's flag.** Keep it local.
- **High-churn values in Context.** Context has no selector; every consumer
  re-renders on any change. Use Zustand (selectors) or local state.
- **A new Context provider per feature.** Prefer a Zustand store unless the value
  is genuinely a tree-scoped capability (auth/role/license).

See also `styles/DESIGN_SYSTEM.md` for visual tokens.
