# Bluefin CLI roadmap

**Last updated:** 2026-10-15

**Current release line:** v0.11.x (v0.11.9 current; v0.10.x in maintenance)

**Planning issue:** [#196](https://github.com/tuna-os/bluefin-cli/issues/196)

Bluefin CLI is a cross-platform environment setup and customization tool in the
[TunaOS](https://tunaos.org) ecosystem. This roadmap describes outcomes and
readiness gates rather than promising individual features or dates. Execution
work should be tracked in linked GitHub issues.

## Current baseline

The project currently ships:

- a persistent native TUI with fuzzy filtering and a command palette;
- multi-shell setup (bash, zsh, fish, ash, nushell), curated application bundles, wallpapers, and Starship themes;
- shared Alpine package manager backend and cross-platform Brewfile workflow;
- runner prompt hang safeguards and PowerShell gsudo privilege boundaries;
- portable setup profiles, self-update, and diagnostics;
- release assets for Linux, macOS, and Windows, including deb and rpm packages;
- CI across Linux, macOS, and Windows, plus scheduled Windows validation.

The release pipeline and direct-download path are operational. **Package-manager
distribution is partially broken** (see near-term section): Homebrew formula publisher is
wired but token-gated; every release since v0.10.6 has silently skipped the tap upload.

## Near term: fix distribution (Q4 2026 critical path)

**Outcome:** every installation path shown as available in the README installs
the current release, and unavailable paths are clearly marked.

**BLOCKED:** Homebrew formula publication silently failed on 14 consecutive releases (v0.10.6 through v0.11.9) due to missing `HOMEBREW_TAP_TOKEN` secret (issue #295). This is a provisioning gap, not a code defect. A maintainer must create a credential and set it as a repository secret before any release will publish the formula.

**Exit gates:**
- [ ] **URGENT (maintainer decision):** Provision `HOMEBREW_TAP_TOKEN` → test release → confirm `Formula/bluefin-cli.rb` lands in tap (related: tuna-os/homebrew-tap provisioning)
- [ ] Publish and verify the Scoop manifest in `tuna-os/scoop-bucket` (same token provisioning path)
- [ ] Bring Winget to the current release line (v0.11.9+) and validate its update loop (currently stuck at v0.8.1)
- [ ] Decide whether Chocolatey and AUR are supported commitments; either publish and verify or keep explicitly out of the supported matrix
- [ ] Add a lightweight post-release check that records which channels received the release and makes partial publication visible

**Exit evidence:** a release checklist links to successful installs or package records for every supported channel, with no stale version claims in the README. v0.11.10+ published to all planned channels.

## Next: define and gate the v1.0 release (Q4/early Q1 2027)

**Outcome:** maintainers and users share a concrete definition of a stable
Bluefin CLI release, and a release candidate passes all gates.

**Depends on:** near-term distribution fixes landing first (issue #295 resolution).

**Exit gates:**
- [ ] Document supported operating systems, architectures, shells, installation methods, and compatibility expectations
- [ ] Define configuration and profile compatibility guarantees, including how breaking changes and migrations are communicated
- [ ] Set release-health expectations: required CI checks, rollback or hotfix ownership, and maximum acceptable lag across supported package channels
- [ ] Resolve or explicitly defer known platform gaps before declaring v1.0
- [ ] Create a v1.0 release checklist issue tracking all gates (see related: issue #223)

**Exit evidence:** a published v1.0-checklist issue is 100% complete, the support matrix is current, and at least one release candidate (v0.12.0-rc.1 or v1.0-rc.1) passes all supported-platform and distribution checks.

## Then: prove adoption and contributor sustainability

**Outcome:** growth and maintenance capacity are visible enough to guide
priorities after v1.

**Exit gates:**
- [ ] Establish a privacy-respecting adoption baseline using release downloads, package-channel installs where available, and opt-in `countme` data
- [ ] Publish a small recurring health snapshot covering active users, release reach, issue response, and external contributions
- [ ] Create contributor-ready issues for bounded documentation, platform verification, and package-maintenance work
- [ ] Document ownership and backup maintainers for release credentials and each supported distribution channel

**Exit evidence:** two consecutive health snapshots are published, at least three contributor-ready issues are maintained, and release/channel ownership does not depend on one person.

## Later opportunities

These remain exploratory until the readiness gates above are met:

- remotely managed or user-defined bundle manifests;
- deeper desktop integration beyond the terminal experience;
- additional curated bundle categories driven by measured demand.

## How this roadmap is maintained

- Review it at each minor release or at least monthly.
- Link execution issues when an outcome moves into active work.
- Move completed outcomes into the baseline instead of leaving shipped work in future-looking sections.
- Record scope changes in the planning issue so roadmap edits remain auditable.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Issues and pull requests are welcome.
