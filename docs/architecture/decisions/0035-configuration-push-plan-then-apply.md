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
| Junos | `configure private`, `load merge\|override terminal`, `show \| compare`, `rollback 0` | device |
| EOS | `configure session <id>`, (`rollback clean-config` for replace), load, `show session-config diffs`, `abort` | device |
| NETCONF | `<edit-config>` into `candidate`, `<validate>`, read back, `<discard-changes>` | device |
| IOS / IOS-XE | none on the device | Seed, against the fresh backup; labelled **predicted** |

An IOS replace predicts the candidate itself as the new configuration. Both
sides are compared without what `show running-config` prints around the
configuration (its banner lines, `!` comments and the final `end`), so a
candidate saved from the device is not a change. A replacement without the
`archive` path is refused at plan time: the next push would need it.

Junos and EOS are driven in an interactive shell, the only way either CLI
reads a load from the session. Replace means the candidate becomes the whole
configuration, which is `load override` on Junos (its `load replace` only
replaces stanzas marked `replace:`) and a session started from
`rollback clean-config` on EOS. The device reads a CLI candidate as typed
input, so a candidate with a control character (`^D` ends a Junos load, `^Z`
leaves EOS configuration mode) is refused, and so is an EOS line that starts
with a command which would end the session or run in exec mode (`commit`,
`end`, `exit`, `abort`, `configure`, `do` and the like). Device errors are
reported by candidate line number and the device's message, without the
configuration text the device quotes back.

The plan stores the pre-push backup id, the running configuration's SHA-256,
the candidate and the diff, with the candidate and diff encrypted under the
credential keyring because both can carry secrets. A plan expires after 15
minutes. An empty diff ends the plan with nothing to apply.

**Apply** — `POST /api/v1/device-config-pushes/{id}/apply` `{planSha256}`,
where `planSha256` is the hash of the diff the admin was shown; another hash
is a 400. A plan is applied at most once: the request claims it, and an
applied or expired plan answers 409. Apply is a job, like plan, because it
takes several logins. The job re-reads the running configuration first
through the backup path; if its SHA-256 differs from the plan's, the job ends
`config_drifted` without touching the device and the admin plans again. Every
claimed apply ends in one recorded outcome: `confirmed`, `rolled_back`,
`reverting` (Seed could neither confirm nor roll back, so the device's timer
reverts), `config_drifted` or `failed`.

### 2. Rollback is the device's job, not Seed's

Apply only uses a mechanism by which the device reverts on its own unless
Seed confirms, so a push that cuts Seed off undoes itself:

| Platform | Apply | Automatic revert |
| --- | --- | --- |
| Junos | `commit confirmed 5` | reverts unless confirmed by `commit` |
| NETCONF | `<commit><confirmed/><persist>id</persist>` | reverts unless confirmed with `<persist-id>` |
| EOS | `configure session <id>`, `commit timer 00:05:00` | reverts unless the session is committed |
| IOS / IOS-XE merge | `configure terminal revert timer 5` (needs `archive`) | reverts unless `configure confirm` |
| IOS / IOS-XE replace | SCP to `flash:<id>.cfg`, `configure replace flash:<id>.cfg force time 5` | as merge |

The EOS session is named after the apply's commit id, so the confirm
(`configure session <id> commit`) and the rollback
(`configure session <id> abort`) from a later login touch only that session.
IOS types the candidate under the revert timer, with each line live as it is
entered; a line the device rejects ends the commit with an immediate
`configure revert now` rather than leaving the earlier lines live until the
timer. An IOS replace copies the candidate to flash as an SCP sink (the device
needs `ip scp server enable`), replaces from it with `force`, which skips the
yes/no question, and requires `Rollback Done`; a replace that fails is reverted
at once if it started the timer. The copy carries the whole configuration,
secrets included, so the same login deletes it whatever the replace did: the
revert restores from the archive, not from the copy. A copy that cannot be
deleted fails the commit, and the timer reverts the replace. EOS and IOS
keep a startup configuration, so the confirm is followed by
`write memory`. A confirmed push whose save fails stays confirmed, and its
outcome says a reload loses the change.

After apply, Seed opens a **new** session, which proves management access
survived, and dry-runs the same candidate again: a device that holds the
planned change has nothing left to change, so an empty diff is the check. Only
then does it confirm, and on the CLI platforms that keep one, save the startup
configuration. The NETCONF commit carries a `<persist>` token so the
confirming commit can come from that new session (RFC 6241 §8.4). If the
post-check fails, Seed rolls back explicitly while it can still reach the
device, and otherwise lets the timer expire. NETCONF rolls back with
`<cancel-commit>` naming the token. Junos rolls back by committing `rollback
1`, which is the configuration before the push only while the pending commit
is the newest, so Seed first reads `show system commit` and leaves the revert
to the timer if someone else has committed since. IOS rolls back with
`configure revert now`, sent only while `show archive config rollback timer`
reports a pending timer. The IOS post-check reads the running configuration
from the new login and predicts the candidate against it, as the plan did
against the backup; a candidate written in another form than the running
configuration's own (an abbreviated command, say) never predicts empty, so it
is rolled back rather than confirmed. A commit that fails or whose result is
unknown is never rolled back by Seed from a later login: undoing "the last commit" then could
undo someone else's, and the timer covers a commit that did land.

A target that lacks the revert mechanism (IOS without `archive`, a NETCONF
server without `:candidate` and `:confirmed-commit:1.1` in its `<hello>`) is
refused at plan time with a named reason. There is no best-effort mode.

**NX-OS is read-only in v1** (owner, 2026-10-16). It has no device-side revert
timer: a checkpoint rollback must be run by Seed, which cannot run it once the
push has cut it off. NX-OS targets keep backup and diff and are refused at plan
time.

### 3. Operating system images are staged, never reloaded automatically

1. The admin uploads an image with the vendor's published hash; an upload
   whose hash does not match is rejected. The request body is the image
   itself, streamed to disk rather than held in memory, and the hash is the
   vendor's hex SHA-256 (Juniper) or SHA-512 (Cisco, Arista); MD5 and SHA-1
   are refused. Seed keeps both digests, since step 4 asks the device for the
   one its platform computes. Images live in `device-images` beside the
   database, named by id, with a record per image; the name is the file name
   the device will get, unique per client. Images are for IOS/IOS-XE, EOS and
   Junos only: NX-OS is read-only and a NETCONF target has no CLI to stage
   with.
2. Seed pre-checks the platform, the running version and image
   (`show version`; EOS names its image in `show boot-config`) and the free
   space on the running image's file system (`dir`). It refuses an image of
   the running image's name, too little free space, and an IOS-XE device
   booted in install mode (`packages.conf`), which takes an image with
   `install add` rather than a boot statement. The copy goes beside the
   running image.
3. Seed copies the image as an SCP sink (`scp -t`) over the existing
   `golang.org/x/crypto/ssh` client, streamed from disk. As with NETCONF, the
   protocol is small enough to write in place, so there is no new dependency.
   EOS's SCP server takes the Linux path (`/mnt/flash/<name>` for
   `flash:<name>`). A copy that fails part way is deleted.
4. The device verifies the copy (`verify /sha512` on IOS and EOS,
   `file checksum sha-256` on Junos). A mismatch deletes the copy.
5. Seed sets the new boot image and keeps the running image as the fallback.
   On IOS/IOS-XE the boot statements become the new image and then the
   running image, are saved, and are read back; statements that cannot all
   be set are put back as they were. EOS boots a single image (`boot system`,
   written to its boot configuration at once): the running image stays on
   flash, and falling back means booting it from Aboot. Seed never deletes the
   running image.
6. Junos runs installed packages, not an image file, so its staging differs
   in three places. The pre-check reads the release from `show version`
   (`Junos: 21.4R3-S5.4`, or the bracketed form before 15.1) and refuses a
   package whose name carries it, since Juniper names a package for its
   release. The package goes to `/var/tmp/`, the file system meant for it,
   and the free space is the `available-blocks` (512-byte blocks) of the
   file system `show system storage | display xml` lists on the longest
   prefix of that path, `/.mount/var` on Junos 21 and later. The boot step is
   `request system software add /var/tmp/<name>` without `reboot`: the device
   validates the running configuration against the new release and unpacks
   it for the next boot, and the step succeeds only when the device reports a
   reboot is required and printed no `error:` or `ERROR:` line. It is given
   30 minutes. The fallback is the software the install keeps for
   `request system software rollback`; the copy stays in `/var/tmp`.

Staging is a job (`POST /api/v1/device-images/{id}/stage` `{targetId}`) that
logs in with the target's push credential and shares the push's one job per
target. It ends `boot_set`, `refused` (a pre-check, before anything was
copied) or `failed`, and records the running version, which the reload checks
the device has left.

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
4. EOS apply, and IOS/IOS-XE merge apply.
5. IOS/IOS-XE replace over the SCP sink (5a); then image upload with the
   vendor hash (5b-1); pre-checks, SCP staging, verification and boot set on
   IOS/IOS-XE and EOS (5b-2) and on Junos (5b-3, §3 step 6); reload as its
   own action (5c).

Tests drive an in-process `x/crypto/ssh` server that scripts each platform's
dialogue, and the apply use-case is driven over a stand-in device through
each outcome: a session dropped after apply (the timer reverts), a
post-check mismatch (explicit rollback), drift between plan and apply
(`config_drifted`), a second apply (409) and a token on apply (403).

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
