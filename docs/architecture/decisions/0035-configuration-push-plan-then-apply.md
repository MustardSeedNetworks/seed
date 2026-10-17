# 0035 — Configuration push: plan, then apply under a device-side revert

Status: Accepted
Date: 2026-10-17

## Context

P-D3 (#3499) asks Seed to push configuration to network devices and to stage
operating system images on them. Everything Seed did to a device before this
was a read: `internal/deviceconfig` logs in over SSH (IOS, NX-OS, EOS, Junos
CLI) or NETCONF with a pinned host key and stores the running configuration,
encrypted, as a backup that P-D2 diffs and alerts on. Seed has no SNMP set, no
configuration-mode SSH command and no SCP client.

A push from a diagnostic appliance is the largest blast radius in the plan: a
wrong line can cut the appliance, and everyone else, off from the device. The
owner set the bounds on 2026-10-04 (admin only, Pro tier, a mandatory dry-run
diff, the running configuration backed up first, rollback on failure, every
push audit-logged) and approved the design on the issue on 2026-10-16 with
three answers: personal access tokens may plan but never apply, a push
targets one device, and NX-OS stays read-only in v1.

## Decision

### 1. Two requests: plan, then apply

No single request can change a device.

**Plan** — `POST /api/v1/device-config-pushes`
`{targetId, mode: "merge"|"replace", candidate}` starts a job that logs in,
takes a fresh backup of the running configuration through the existing backup
path (so it is stored, encrypted and diffable like any other backup), and
asks the device for the dry-run diff:

| Platform | Dry run | Diff source |
| --- | --- | --- |
| Junos | `configure private`, `load merge\|replace terminal`, `show \| compare`, `rollback 0` | device |
| EOS | `configure session <id>`, load, `show session-config diffs`, `abort` | device |
| NETCONF | `<edit-config>` into `candidate`, `<validate>`, read back, `<discard-changes>` | device |
| IOS / IOS-XE | none on the device | Seed, against the fresh backup; labelled **predicted** |

The plan stores the pre-push backup id, the running configuration's SHA-256,
the candidate and the diff, with the candidate and diff encrypted under the
credential keyring because both can carry secrets. A plan expires after 15
minutes. An empty diff ends the plan with nothing to apply.

**Apply** — `POST /api/v1/device-config-pushes/{id}/apply` `{planSha256}`,
where `planSha256` is the hash of the diff the admin was shown. Seed re-reads
the running configuration first; if its SHA-256 differs from the plan's, apply
answers `409 config_drifted` and the admin plans again. A plan is applied at
most once.

### 2. Rollback is the device's job, not Seed's

Apply only uses a mechanism by which the device reverts on its own unless
Seed confirms, so a push that cuts Seed off undoes itself:

| Platform | Apply | Automatic revert |
| --- | --- | --- |
| Junos | `commit confirmed 5` | reverts unless confirmed by `commit` |
| NETCONF | `<commit><confirmed/><persist>id</persist>` | reverts unless confirmed with `<persist-id>` |
| EOS | `configure session <id>`, `commit timer 00:05:00` | reverts unless the session is committed |
| IOS / IOS-XE | `configure terminal revert timer 5` (needs `archive`) | reverts unless `configure confirm` |

After apply, Seed opens a **new** session, which proves management access
survived, re-reads the running configuration and checks that the planned
change is present. Only then does it confirm, and on the CLI platforms save
the startup configuration. The NETCONF commit carries a `<persist>` token so
the confirming commit can come from that new session (RFC 6241 §8.4). If the
post-check fails, Seed rolls back explicitly while it can still reach the
device, and otherwise lets the timer expire.

A target that lacks the revert mechanism (IOS without `archive`, a NETCONF
server without `:candidate` and `:confirmed-commit:1.1` in its `<hello>`) is
refused at plan time with a named reason. There is no best-effort mode.

**NX-OS is read-only in v1** (owner, 2026-10-16). It has no device-side revert
timer: a checkpoint rollback must be run by Seed, which cannot run it once the
push has cut it off. NX-OS targets keep backup and diff and are refused at plan
time.

### 3. Operating system images are staged, never reloaded automatically

1. The admin uploads an image with the vendor's published hash; an upload
   whose hash does not match is rejected.
2. Seed pre-checks the platform, the running version (`show version`) and the
   free flash space.
3. Seed copies the image as an SCP sink (`scp -t`) over the existing
   `golang.org/x/crypto/ssh` client. As with NETCONF, the protocol is small
   enough to write in place, so there is no new dependency.
4. The device verifies the copy (`verify /sha512` on IOS and EOS,
   `file checksum hash sha-256` on Junos). A mismatch deletes the copy.
5. Seed sets the new boot image and keeps the running image as the fallback
   boot entry. Seed never deletes the running image.

Reloading is a separate admin action with its own confirmation. After the
reload, Seed checks the device is back on the expected version. Seed cannot
undo a bad image remotely: the fallback boot entry is the rollback, and the
reload confirmation says so.

### 4. Authorization

- Plan, apply, image upload, staging and reload are `Scope: roles.Admin` with
  `Auth` and `CSRF`, behind a new Pro feature, `config_push`.
- **Apply and reload refuse a personal access token** (`auth.IsAPITokenAuth`)
  with 403, so a leaked token can prepare a change but never make one. A plan
  may be created by token.
- Reading plans and push history is operator and above, the same rule as
  reading backups: a diff carries configuration text.
- Pushing is off per target by default. An admin enables it by giving the
  target a push credential (`PUT /device-config-targets/{id}/push`
  `{credentialsId}`; `DELETE` turns it off), which must be another credential
  than the backup's, so the read-only backup login never gains write access.
  The target carries it as `pushCredentialsId`, matching its `credentialsId`;
  a separate `allowPush` flag would only add a state with a flag and no
  credential. Editing the target keeps it.
- A push needs a pinned host key; an unpinned or changed key refuses the plan,
  exactly as a backup does.
- One push or image job runs per target at a time. A push names one target;
  there is no group or bulk push in v1.

### 5. Audit

`RecordAuditLog` records plan created, plan refused, apply started, confirmed,
rolled back (explicitly or by timer), image uploaded, staged, verified, boot
set, reload requested and reload verified. Each entry carries the user, IP,
target, plan or image id, diff SHA-256, added and removed line counts and the
outcome. Configuration text never enters the audit log or an alert, the P-D2
rule. A completed push stores a normal backup, so P-D2's `config.change`
alert fires on the new configuration.

### 6. Delivery

1. This ADR.
2. Migration, the per-target push credential, the `config_push` feature,
   and the plan endpoint (dry run only; nothing is written to a device), for
   NETCONF and IOS/IOS-XE; then the Junos and EOS dry runs, which are refused
   at plan time until they land.
3. Apply, confirmed commit, post-check and rollback for Junos and NETCONF.
4. EOS and IOS/IOS-XE.
5. SCP image staging, verification and boot set; reload as its own action.

Tests drive an in-process `x/crypto/ssh` server that scripts each platform's
dialogue, including a session dropped after apply (the timer reverts), a
post-check mismatch (explicit rollback), drift between plan and apply (409)
and a token on apply (403).

## Consequences

- Seed can change a customer network for the first time. The blast radius is
  bounded to one device per admin action, a change that strands Seed reverts
  on the device, and the change is visible beforehand as a diff and afterwards
  in the audit log and the change alert.
- IOS needs `archive` configured before it can be pushed to; the plan refusal
  names that, and the setup document gives the lines.
- The IOS diff is a prediction, because the device offers no preview. The
  post-check is what proves the result, and the UI labels the prediction.
- NX-OS push, bulk or group push, scheduled push, templates and variables,
  automatic reload and SNMP set are not in v1. Each needs its own row and
  design.
- No real device is in the lab. Real-hardware validation per platform is an
  owner step, like S4-3, and is not claimed by any slice's acceptance.
