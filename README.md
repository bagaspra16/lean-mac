# lean-mac

**Developer-aware storage analysis & cleanup for macOS — entirely in your terminal.**

Binary: **`lm`**. Apple Silicon native. No daemons, no menubar apps, no telemetry.

If you're a developer on a 256 GB or 512 GB Mac, you've watched your disk fill up with `node_modules`, Docker layers, Xcode caches, and package-manager artifacts you no longer need. `lm` finds them, explains them, and — only with your explicit confirmation — removes them.

---

## Why this exists

Stock macOS "Storage" shows you which **apps** are big. It can't tell you that 6 GB of your "system data" is shutdown iOS simulators, that another 5 GB is `node_modules` directories you haven't touched in a year, or that Docker is sitting on 8 GB of dangling build cache. `lm` knows about the specific places developer tools hide data, and treats each one with a risk level you can trust.

It is not a generic "Mac cleaner". It does one thing: it is **operationally honest about where dev-tool storage goes**, and gives you a safe, scriptable way to reclaim it.

---

## Feature tour

### 🔍 Concurrent scanner with 18+ detectors

Walks your home directory and queries native tools concurrently. One scan finds artifacts across:

| Category                  | Source                                          |
| ------------------------- | ----------------------------------------------- |
| `node_modules`            | every `package.json` project under `$HOME`      |
| `rust-target`             | every `Cargo.toml` project                      |
| `docker-images`           | `docker system df`                              |
| `docker-volumes`          | `docker system df`                              |
| `docker-buildcache`       | `docker system df`                              |
| `ios-simulators`          | `xcrun simctl list` (per shutdown device)       |
| `xcode-deriveddata`       | `~/Library/Developer/Xcode/DerivedData`         |
| `xcode-archives`          | `~/Library/Developer/Xcode/Archives`            |
| `npm-cache`, `pnpm-store` | `~/.npm`, `~/Library/pnpm/store`                |
| `yarn-cache`, `pip-cache` | `~/Library/Caches/Yarn`, `~/Library/Caches/pip` |
| `go-cache`, `go-modcache` | `~/Library/Caches/go-build`, `~/go/pkg/mod`     |
| `cargo-cache`             | `~/.cargo/registry`                             |
| `gradle-cache`            | `~/.gradle/caches`                              |
| `maven-cache`             | `~/.m2/repository`                              |
| `homebrew-cache`          | `~/Library/Caches/Homebrew`                     |

A scan over a typical developer home directory completes in 10–30 seconds and reports total reclaimable space, grouped and ranked by size.

### 🎚 Risk-aware cleanup

Every finding carries one of three labels:

- 🟢 **SAFE** — caches that auto-regenerate. Nothing breaks. *(npm cache, Homebrew cache, Xcode DerivedData, etc.)*
- 🟡 **MEDIUM** — costs time / bandwidth to rebuild. *(Go module cache, iOS simulators, cargo registry.)*
- 🔴 **DANGEROUS** — may contain user data or shipped artifacts. *(Docker volumes, Xcode Archives.)*

`lm clean` only acts on SAFE by default. `--aggressive` includes MEDIUM. `--dangerous` is required for DANGEROUS. There is no flag that defeats the protected-path list.

### 🤖 AI Cleanse — conversational cleanup (optional)

Press `2` in the TUI to chat with an LLM (Groq-hosted Llama 3.3 70B by default). It will:

1. Run a scan on your behalf.
2. Explain in plain English what's eating your disk.
3. Propose deletions **one category at a time**.
4. Wait for you to press `y` (approve), `n` (reject), `a` (auto-approve future SAFE proposals), or `c` (cancel) on each step.

The AI can never specify file paths. Its only available tool calls are `scan_disk()` and `propose_cleanup(category)`, where `category` is an enum of the 18 known detector names. Anything outside that list is rejected before the cleaner is invoked. The protected-path check in the cleaner is the final line of defense.

AI Cleanse is **bring-your-own-key**: get a free Groq API key, drop it in `~/.config/lean-mac/config.toml`, and the AI tab lights up. Without a key, the AI tab shows setup instructions and the rest of `lm` works as before.

### 💻 Terminal-first interface

Three tabs in the TUI, switchable with `1` / `2` / `3` or `tab`:

| Tab            | What it does                                                                       |
| -------------- | ---------------------------------------------------------------------------------- |
| **Scan**       | Live findings table, grouped by category. Mark with `space`, delete with `d`.      |
| **AI Cleanse** | Conversational cleanup with per-action approval.                                   |
| **Help**       | Scrollable glossary: every category explained, every keybind, safety guarantees.   |

A persistent header shows the version, disk usage with an inline bar, the current view title, and contextual status. A footer always shows the keys available *right now* for what you're doing.

### 📊 Scriptable CLI

The TUI is optional. Everything `lm` can do, it can do as a one-shot CLI:

```sh
lm scan                                       # text summary
lm scan --json                                # machine-readable
lm clean --dry-run                            # preview what would be removed
lm clean --only npm-cache,xcode-deriveddata   # surgical
lm clean --aggressive --yes                   # include MEDIUM, skip prompts
lm report --format markdown --out report.md   # write a report file
lm monitor                                    # live disk-pressure stream
lm doctor                                     # check environment
```

Use it in a cron job, a CI step, or a `~/bin/morning-cleanup.sh`.

---

## Install

### Via Homebrew (recommended)

```sh
brew install bagaspra16/lean-mac/lm
```

That's all. The tap and formula are public. `brew upgrade lm` updates you when a new tag ships.

### From source

```sh
git clone https://github.com/bagaspra16/lean-mac.git
cd lean-mac
make install
```

`make install` builds `bin/lm` and installs it to `/opt/homebrew/bin/lm` (already on `$PATH` for Apple Silicon Homebrew). `make uninstall` reverses it. Requires Go 1.22+.

### Requirements

- macOS, Apple Silicon. Intel may work — untested.
- Optional but useful: `docker`, `xcrun`, `brew` — `lm doctor` will tell you which are available.

---

## Using `lm`

### The interactive TUI

```sh
lm
```

Press `?` at any time for the in-app Help glossary. The keys you need to know:

```
Global
  tab / shift+tab     switch tab
  1 / 2 / 3           jump to a tab
  ?                   open Help
  q / ctrl+c          quit

Scan tab
  j / k               move cursor
  g / G               top / bottom
  space               mark item
  a                   mark every SAFE item
  /                   filter (enter/esc to leave)
  d                   delete marked (with confirm)

AI Cleanse tab
  type, enter         send a message
  y / n               approve / reject the current proposal
  a                   auto-approve future SAFE proposals only
  c                   cancel the agent
```

The TUI runs in **dry-run mode by default** — even when you press `d` and confirm, no files are deleted unless you explicitly started with live mode. This is intentional. Use the CLI for live runs.

### The CLI

```sh
lm scan
```

Prints a category-grouped summary of reclaimable space. Read-only — touches nothing.

```sh
lm clean
```

Scans, then asks per category whether to proceed. SAFE-risk findings are eligible by default. Add `--aggressive` for MEDIUM, `--dangerous` for DANGEROUS.

Flags:

```
--dry-run        simulate; report what would be removed, then exit
--yes            skip the per-category prompt (for scripts)
--only LIST      limit to comma-separated categories
--aggressive     include MEDIUM risk findings
--dangerous      include DANGEROUS findings (extra prompt anyway)
--json           emit JSON
```

Example pipelines:

```sh
# Inspect first
lm scan --json | jq '.findings | group_by(.category)'

# Reclaim known-safe space, no prompts, in a script
lm clean --only npm-cache,homebrew-cache,xcode-deriveddata --yes

# Quarterly "deep clean" with a record of what happened
lm clean --aggressive --yes | tee ~/Documents/lm-$(date +%F).log
```

### Reports

```sh
lm report --format json     --out scan.json
lm report --format markdown --out scan.md
```

Drop `--out` to print to stdout. Useful for handing the output to teammates or attaching to a ticket.

---

## Setting up AI Cleanse

The AI feature is optional. You bring your own Groq API key (free tier is generous — no credit card required at the time of writing).

1. Get a key at **https://console.groq.com/keys**. Click *Create API Key*, copy the value starting with `gsk_`.

2. Save it to a config file:

   ```sh
   mkdir -p ~/.config/lean-mac
   cat > ~/.config/lean-mac/config.toml <<EOF
   groq_api_key = "gsk_paste_yours_here"
   EOF
   chmod 600 ~/.config/lean-mac/config.toml
   ```

   Or use environment variables:

   ```sh
   export GROQ_API_KEY=gsk_...
   # Up to 9 are read (GROQ_API_KEY_2 … GROQ_API_KEY_9); lm rotates between
   # them on rate-limit (429) errors.
   ```

3. Restart `lm`. The AI tab now greets you with a welcome card and sample prompts.

The key never leaves your machine except over HTTPS to `api.groq.com`. It is not logged, not stored anywhere else, and not bundled in the binary.

---

## Safety guarantees

These are not aspirational — they are properties of the code, enforced regardless of flags or AI input:

1. **Dry-run is the default in the TUI.** The interactive view never executes a real deletion in this version. Use the CLI for live runs.
2. **Protected paths are refused at the syscall level.** The cleaner will not delete `/`, `/System`, `/Library`, `/usr`, `/bin`, `/sbin`, `/etc`, `/var`, `/Applications`, `/Users`, or your home directory itself. No flag overrides this — it's hard-coded.
3. **The AI cannot name paths.** Its tool schema accepts only a category enum. A model that hallucinates `/etc/passwd` will get a schema error, not a deletion.
4. **Native tools, not raw `rm`.** Docker objects go through `docker prune`, simulators through `xcrun simctl delete` — same commands you'd run by hand.
5. **Per-action approval in the AI flow.** Even with `a` (auto-approve), only SAFE proposals are auto-approved. MEDIUM and DANGEROUS always require explicit `y`.

---

## Project layout

```
cmd/lm/                 entrypoint
internal/
  types/                Finding, ScanReport, Risk, Category
  fsutil/               cancellable directory sizing, statfs disk usage
  scanner/              Detector interface + concurrent runner
  detectors/            18 detectors: node, rust, docker, simulators, paths
  cleaner/              risk gating, protected-path enforcement, dispatch
  ai/                   Groq client, agent loop, tool-call safety
  config/               env + ~/.config/lean-mac/config.toml loader
  monitor/              disk-pressure streaming
  reporting/            text / JSON / Markdown writers
  cli/                  cobra subcommands
  ui/                   Bubble Tea TUI: app chrome, scan / ai / help views
Formula/lm.rb           Homebrew formula (also mirrored to the tap repo)
scripts/                release helpers
```

---

## Releasing (for the maintainer)

```sh
make tag VERSION=v0.4.0                  # tags + pushes
./scripts/update-formula-sha.sh v0.4.0   # rewrites Formula/lm.rb sha256
git add Formula/lm.rb && git commit -m "formula: v0.4.0" && git push

# Then mirror the formula to the tap repo:
cd /path/to/homebrew-lean-mac
git pull
cp ../lean-mac/Formula/lm.rb Formula/lm.rb
git add Formula/lm.rb && git commit -m "tap: bump to v0.4.0" && git push
```

Users pick up the new version with `brew update && brew upgrade lm`.

---

## Contributing

I'm happy to take help. There's plenty to build (see "Roadmap" below).

**Before you open a PR, please email me first** at **bagaspratamajunianika72@gmail.com** with a quick note about what you want to work on. This isn't gatekeeping — it's so neither of us wastes time. I want to make sure:

- the feature fits the project's stance (developer-aware, safe by default, terminal-first),
- nobody else is already mid-flight on the same thing,
- we agree on the design before code gets written.

After that, the loop is normal: fork, branch, PR. CI runs `go vet` and `go build`. Please keep commits focused and write commit messages that explain *why*.

If you find a bug or have an idea but don't want to implement it, an issue is welcome — same email or open one on GitHub.

### Roadmap (deliberately not implemented yet)

These are good places to start if you want to contribute:

- Scheduler / launchd integration (`lm` running periodically with a sensible default profile).
- Duplicate-file detection across the home directory.
- iPhone backup, Time Machine snapshot, browser cache detectors.
- HTML report output.
- External SSD migration assistant.
- Plugin system so third parties can ship their own detectors.
- Profile presets (`safe-clean`, `aggressive-clean`, `frontend-dev`, etc.) instead of just flags.

The architecture is built for them: a new detector is one file implementing `scanner.Detector`, and a new cleanup pathway is one case in `cleaner.Cleaner.cleanOne`.

---

## License

MIT. Use it, modify it, ship it. Attribution appreciated but not required.

---

## Author

Built by [@bagaspra16](https://github.com/bagaspra16). Reach me at **bagaspratamajunianika72@gmail.com**.
