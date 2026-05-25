# lean-mac

Developer-aware storage analysis & cleanup for macOS. Terminal-first.

Binary: **`lm`**.

This is a working core, not the full spec. What ships today:

- Concurrent scanner with ~16 detectors: `node_modules`, Rust `target/`,
  Docker images / volumes / build cache, iOS simulators (per-device),
  Xcode DerivedData & Archives, npm / pnpm / yarn / pip / Homebrew / Gradle /
  Maven / cargo / Go build & module caches.
- Risk classification per finding (`SAFE`, `MEDIUM`, `DANGEROUS`).
- Risk-aware cleaner: `SAFE` always eligible, `MEDIUM` requires `--aggressive`,
  `DANGEROUS` requires `--dangerous`. Protected paths refused at the syscall.
- CLI: `scan`, `clean`, `report`, `monitor`, `doctor`, `tui`.
- Bubble Tea TUI with live findings, grouped table, risk colors, search,
  per-row marking, confirmation modal.
- JSON / Markdown / text report writers.

## Install

**On this machine** (Apple Silicon, Homebrew prefix already on `$PATH`):

```
make install
```

That builds `bin/lm` and copies it to `/opt/homebrew/bin/lm`. Run `lm` from any directory after that. `make uninstall` removes it.

**Via Homebrew** (once the tap is published):

```
brew tap bagaspra16/lean-mac
brew install lm
```

The formula lives at `Formula/lm.rb` and is meant to be mirrored into a tap repo named `homebrew-lean-mac` under the same GitHub user.

### Cutting a release (for the maintainer)

```
make tag VERSION=v0.1.0                     # tag + push
./scripts/update-formula-sha.sh v0.1.0      # rewrite url+sha in Formula/lm.rb
# then commit Formula/lm.rb into the homebrew-lean-mac tap repo
```

Requires Go 1.22+. No CGo. Apple Silicon native.

## Use

```
lm              # launch TUI
lm scan         # text summary, no deletion
lm scan --json  # machine-readable
lm clean        # scan, then interactive per-category confirm
lm clean --dry-run --yes --only npm-cache,xcode-deriveddata
lm monitor      # live disk-pressure stream
lm doctor       # check environment
```

`clean` is interactive-confirm by default. Pass `--yes` to skip prompts
(e.g. in scripts). `--dry-run` simulates without touching anything.

## Safety

- The cleaner refuses to remove `/`, `/System`, `/Library`, `/usr`, `/bin`,
  `/sbin`, `/etc`, `/var`, `/Applications`, `/Users`, or the user's home
  directory itself.
- Docker objects are pruned via `docker` CLI; iOS simulators via
  `xcrun simctl delete`. Filesystem objects use `os.RemoveAll` on the exact
  path the detector emitted.
- Risk gating means a single `lm clean` without flags will only act on
  `SAFE` findings (caches that regenerate).

## Not yet implemented

Items from the original spec deliberately deferred:

- Scheduler / cron-style automation.
- Duplicate file detection.
- iPhone backup, Time Machine snapshot, browser cache detectors.
- HTML report.
- External SSD migration / cloud offload.
- Plugin loading for third-party cleaners.
- Profile system (`safe-clean`, `aggressive-clean`, etc. — currently flags only).

The architecture supports them: add a `scanner.Detector` and a corresponding
case in `cleaner.Cleaner.cleanOne`.
