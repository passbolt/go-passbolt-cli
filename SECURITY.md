# Security Policy

For general information about security at Passbolt, see https://www.passbolt.com/security

## Reporting a security issue

If you've found a security issue in Passbolt, please do not open a public issue. Follow our responsible disclosure process: https://www.passbolt.com/docs/contribute/security/vulnerability/

## Where to store the config file

The config file holds your private key, its passphrase and your MFA seed in cleartext. Anyone who can read it can log in as you and decrypt every password you can see.

### What the CLI does

- It creates the config file and the KeePass export readable by you alone: mode `0600`, or on Windows an ACL granting only you, SYSTEM and Administrators[^windows-acl].
- It tightens a file whose permissions are too loose, then checks that the change took effect.
- If the permissions stay loose (on a disk that keeps none, for instance), or the file belongs to another user (as a network or VM shared drive often reports), it writes the file anyway and prints a warning to stderr, so output on stdout is unaffected.
- It refuses to write to something that is not a regular file, such as a device or a pipe.

[^windows-acl]: SYSTEM is Windows itself, and an administrator can take ownership of any file whatever its ACL, so leaving them out would protect nothing. It is also the default ACL of your profile folder.

### What it does not do

File permissions only stop other users of **this machine**, and only on a disk that enforces them. The CLI does not check where the file is stored; that choice is yours.

Permissions do not protect the file in these places:

| Place | Who else can read it |
|---|---|
| USB sticks and external drives (FAT, exFAT) | anyone who plugs the drive in; these filesystems keep no permissions |
| Network shares (SMB, NFS) | the file server's administrators, and often other users of the share |
| VM or container shared folders | the host, and anything else with access to the shared folder |
| Cloud-synced folders (Dropbox, OneDrive, iCloud, ...) | the sync provider, and any device or person the folder is shared with |
| Backups and disk images | anyone who can read the backup |

### What to do

- Keep the file on a local disk, in your home directory. The default location is fine.
- If that is not possible, save only `serverAddress` and pass the rest through environment variables or flags. With no `userPassword` saved, the CLI asks for it each time.

## Opt-in authentication agent

The agent is disabled by default. Enabling it keeps an unlocked account key and
Passbolt session/MFA cookies in a background process until its one-hour idle
lease expires or you lock/stop it. Each authenticated CLI invocation requires a
separate approval; status queries and rejected requests do not extend the lease.
Resource reads via `get resource`, `exec` environment substitution, and credential helpers additionally require
approval of the resource name and UUID before fetching its secret. Requester
executable names and parent PIDs are best-effort UI context, not an authorization
boundary. Command-line arguments are never inspected for display.
A valid server session may expire sooner and server MFA requirements remain in
force. A command already approved may finish after its lease deadline.

IPC uses an owner-only Unix socket in a checked `0700` directory, or a local-only
Windows named pipe restricted to the Windows logon SID. The Windows client
verifies the connected server's logon identity before transmitting configuration.
The request protocol is bounded and versioned. Authentication material is not
returned to foreground clients, placed in launch arguments/environment, logged,
or persisted. An `exec` request intentionally returns the resource values needed
for the requested child program's environment.

This protects against other ordinary local users, not administrators, debuggers
with access to your process, or a compromised account running as you. The agent
receives its UI/session context from the invoking same-user CLI. Approval identifies
the server, account fingerprint and command path; it does not list argument values,
which may themselves contain secrets. OS swap, crash dumps and managed-runtime
copies can retain memory; this feature does not promise locked RAM or forensic
memory erasure. Mutable protocol/input buffers and SDK key material are cleared
where supported, including offline local lock. Stopping the process releases its
address space.

The first implementation serializes commands. A stalled command can delay other
requests and expiry cleanup until it finishes; connection cancellation and request
timeouts cancel its network operations. A lost connection is never automatically
retried because a server write may already have completed.

## Machine credential outputs

Raw extraction, Git/Docker helpers, AWS credential_process and Kubernetes exec
reserve stdout for the selected protocol. Prompts use a separate desktop/TTY UI;
diagnostics go to stderr. Debug logging is rejected. Server-controlled lookup and
session-key cleanup errors are redacted/suppressed in machine modes. Docker's
fixed upstream not-found marker is protocol data, not an arbitrary diagnostic.

Git matches service URI scheme, authority and path metadata, rather than treating
a URL as a Passbolt resource selector. Docker matches registry origins. Searches
do not decrypt passwords; only a selected immutable resource ID is read, with
current URI/account metadata revalidated before release. Ambiguous unattended
lookups cancel even if a desktop is present. Interactive selection is explicit
and cannot be bypassed with --yes. Docker list reports usernames only and never
reads resource passwords. Store/erase operations never alter resources.

Raw, AWS and Kubernetes outputs use explicit resource UUIDs. Returned credentials
are intentionally released only to the requesting consumer over stdout, never
written to configuration or injected into environment variables. Raw mode trims
surrounding whitespace; Git/Docker retain values within their standard formats.
The official Docker runtime handles its wire protocol; Kubernetes uses its
upstream clientauthentication/v1 structure. AWS optional fields use nil pointers
when absent. Expiration values are validated before any output.

Cloud process invocations remain subject to user consent and server policy.
Kubernetes' standard exec request can prohibit interaction; the helper honors that
restriction rather than prompting through a hidden terminal or GUI. Memory and
same-user limitations described for the agent also apply to these helper modes.
