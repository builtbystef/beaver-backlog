---
name: set-up-sandbox
description: Sets a machine up with agentbox — a single bwrap wrapper that runs any coding-agent CLI (Claude Code, Pi, Codex) with the host environment intact except that private directories and credential stores are masked, personal data is read-only, and sudo is dead. One machine-level policy, no per-repo config, no per-command sandbox.
disable-model-invocation: true
---

# Set Up Sandbox (agentbox)

One wrapper script, `agentbox`, runs any agent CLI inside a single outer bwrap namespace. The agent — its shell commands *and* its native file tools, the whole process tree — sees the real machine except for an explicit deny list. Nothing else changes: network, `/tmp`, unix sockets, localhost services, browsers, git all behave exactly as unsandboxed.

The security model:

> Private directories (`~/Main` for this machine's owner) and credential stores are unreachable — masked as empty at the OS level. Personal data directories, shell startup files, and the policy script itself are read-only. `sudo`/`su`/`pkexec` are dead (bwrap mounts everything nosuid), so the OS itself cannot be damaged. Everything else — including the rest of `$HOME` — is readable and writable, and an hourly root-owned rsync hardlink snapshot bounds the damage a destructive or compromised agent can do to it. Repositories are additionally backed by their pushed remotes.

## The script

[assets/agentbox](assets/agentbox) — install to `~/.local/bin/agentbox`, `chmod +x`. Usage: `agentbox claude`, `agentbox pi`, `agentbox codex`, `agentbox bash` (arbitrary commands for testing). Invocation is explicit — no shell aliases; the user types `agentbox claude` when they want the wrapper.

What it does, in order:

- `--dev-bind / /` — the whole host, then subtract.
- **Masks** (`--tmpfs` dirs, `--bind /dev/null` files) over: `~/Main`, `~/.ssh`, `~/.gnupg`, `~/.local/share/keyrings`, the browser profiles (`~/.config/google-chrome`, `~/.config/chromium`, `~/.config/BraveSoftware`, `~/snap/firefox`), the agent sockets `/run/user/<uid>/keyring` and `/run/user/<uid>/gnupg`, and `/run/docker.sock`.
- **Per-agent credential complement**: each agent sees only its own auth file — `agentbox claude` masks Pi's and Codex's tokens, and vice versa. An agent's *own* token is inherently available to it (it authenticates with it); that exposure cannot be configured away.
- **Read-only** (`--ro-bind`): `~/dotfiles`, the personal data dirs (`~/Music`, `~/Pictures`, `~/Documents`, `~/Videos`, `~/Desktop`), the shell startup files (`~/.bashrc`, `~/.profile`, `~/.zshenv`), `~/.claude/settings.json`, `~/.config/gh` (usable, not swappable — see below), and the agentbox script itself.
- Every mask/ro helper skips paths that don't exist — bwrap would otherwise create the mount point on the **real** filesystem and leave it behind (measured 2026-08-23). Adapt the lists to the machine, but only ever list existing paths.

Useful properties that fall out for free (all verified live 2026-08-27):

- **sudo is dead** by nosuid — no command deny-list to maintain or bypass.
- **Agents cannot ssh**: `~/.ssh` masked kills ssh-based git remotes and any other ssh use.
- **`gh` works inside** — see "GitHub CLI" below; the token's scopes, not the wrapper, bound remote actions.
- **ssh-agent/gpg-agent are unreachable** — `ssh-add -l` inside must fail with `Error connecting to agent: No such file or directory` (that exact error is the mask working).
- **`~/dotfiles` stays readable** so rc files (symlinked into it) load normally inside — verified to contain no secrets before choosing ro over mask. Re-check if secrets ever land there.
- A write into a masked dir "succeeds" into the tmpfs and evaporates; a masked dir lists as empty. Both are the mask working, not a bug.

## GitHub CLI (optional)

If the user wants agents to use `gh` (view/create/merge/close PRs), have them log it in with a **fine-grained PAT**, not their broad OAuth token:

1. github.com → Settings → Developer settings → Fine-grained tokens: scope it to the repos agents work on, permissions **Contents** and **Pull requests** read/write, **Metadata** read — **no** Administration, no Workflows.
2. `gh auth login --with-token --insecure-storage` (paste the PAT). `--insecure-storage` is required: the keyring is masked inside the wrapper, so the token must live in `~/.config/gh/hosts.yml`.

Agents then can't delete repos, change settings, manage keys/secrets, or touch workflow files. `~/.config/gh` stays ro-bound so they can't log in a broader token.

## Verify after install (and after any bwrap/OS upgrade)

```sh
agentbox bash -c '
ls -A ~/Main | wc -l                      # 0
cat ~/.ssh/config                         # No such file or directory
gh auth status                            # logged in (fine-grained PAT)
touch ~/.config/gh/x                      # Read-only file system
sudo -n true                              # fails
curl -sS -o /dev/null -w "%{http_code}\n" https://google.com   # works: no net namespace
touch ~/Music/x                           # Read-only file system
touch ~/.local/bin/agentbox               # Read-only file system
docker ps                                 # cannot connect (rootful socket masked)
ssh-add -l                                # Error connecting to agent
'
```

Also confirm the agent authenticates end-to-end: `agentbox claude -p "say ok"`. When testing DNS, use a domain known to resolve on this network — this machine's router NXDOMAINs `example.com` (measured 2026-08-27; cost an hour of false debugging).

Ubuntu 24.04+ needs an AppArmor bwrap/userns fix. Unload `bwrap-userns-restrict`, install a `flags=(unconfined)` profile for `/usr/bin/bwrap` with `userns,`). Without it bwrap fails closed.

## Accepted risks — say them plainly in the handover

- **Open egress + open reads**: a prompt injection in anything the agent reads can exfiltrate anything readable (other repos, shell history, transcripts). The masks bound that to non-credential, non-private data; secrets on this machine are throwaway by policy (production secrets live in a secret manager).
- **Persistence is possible**: with writes open, an agent can drop binaries into writable PATH dirs (`~/.local/bin` beside agentbox, `~/.bun/bin`, nvm dirs). The ro-binds on rc files and the script close the lazy paths, not all paths. Backups plus root-owned system files are the backstop, not prevention.
- **The wrapper only protects what it wraps**: bare `claude` runs naked. Invocation is deliberately explicit (no aliases) — the user must type `agentbox claude` and owns that discipline.
- Docker: the rootful socket is masked because socket access is root on the host. For container work, run the daemon in a VM (Docker Desktop for Linux with file sharing limited to `~/Code` and `/tmp`) — then `docker` works inside agentbox with the VM as the blast-radius boundary, and the rootful daemon stays disabled.
- Inside the wrapper, run agents in permissive modes freely (`--permission-mode acceptEdits`, or skip-permissions for zero-prompt): the boundary is outside the agent, where the agent cannot edit it. This deliberately supersedes the old "never recommend bypass flags" rule, which belonged to the abandoned model where the agent's own config was the boundary.

## Backups (the damage bound)

rsync hardlink snapshots (a `backup-home` script), hourly, to a **root-owned** `/var/backups/<user>` (mode 700) via a root systemd timer — the agent cannot delete what it cannot write, and sudo is dead inside the wrapper. Exclude `.cache`, `node_modules`, build artifacts, `snap`, and the masked dirs (they'd snapshot as empty). Same-disk: protects against agent damage, not disk failure.
