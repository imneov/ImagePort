# Security policy

ImagePort is in initial development. There are no supported production releases
yet. Until a stable release policy is published, security fixes target `main`.

## Reporting a vulnerability

Use GitHub's **Report a vulnerability** action on this repository's Security tab
when private vulnerability reporting is enabled. If it is unavailable, open a
minimal issue requesting a private reporting channel, without exploit details,
credentials, private registry names or sensitive artifacts. A maintainer will
provide a private channel before you share details.

Please include affected revisions, platform, reproduction steps and impact in
the private report. Never upload passwords, tokens, authentication headers,
private image layers or unredacted diagnostic captures.

## Security boundaries

- Registry profiles and task records contain metadata, not passwords or tokens.
- Persistent credentials belong in the operating system's keyring; session-only
  credentials are opt-in and disappear when the application exits.
- TLS certificate verification is required for remote registries. Custom CAs are
  scoped to the selected profile. HTTP is loopback-only for development.
- Archives are untrusted input: paths, links, resource limits, referenced content
  and available digests must be checked before transfer.
- Local publication must not replace existing files. Remote replacement requires
  explicit target consent and a fresh conflict check.
- Cancellation is not rollback, and digest equality is not a signature or proof
  of image origin.

Before publishing a release, maintainers must check dependencies, native platform
behavior, credential handling and the acceptance matrix. Configure repository
security settings through GitHub; files in this repository do not claim those
server-side settings have been enabled.
