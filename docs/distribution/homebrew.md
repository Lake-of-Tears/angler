# Homebrew Distribution Pipeline & Roadmap

The official Homebrew distribution pipeline for `angler` is organized into a two-phase strategy:

1. **Immediate Distribution via Official Tap (`Lake-of-Tears/homebrew-tap`)**: Zero-friction automated installation and updates on macOS and Linux via `brew install Lake-of-Tears/tap/angler`.
2. **Upstream Homebrew Core Roadmap (`homebrew/core`)**: Source-build formula specification and tracking criteria for upstream submission into Homebrew's default core repository.

---

## Phase 1: Official Tap (`Lake-of-Tears/tap`)

### Installation

```bash
# Tap the official repository and install
brew tap Lake-of-Tears/tap
brew install angler

# Or install in a single command
brew install Lake-of-Tears/tap/angler
```

Shell completions for **Bash**, **Zsh**, and **Fish** are automatically generated and linked during installation using Homebrew's `generate_completions_from_executable` hook.

### Upgrades

```bash
brew update
brew upgrade angler
```

### Automation & CI/CD

- **Release Workflow**: Automated via `.github/workflows/release.yaml` on manual dispatch or semver tags.
- **GoReleaser Integration**: Configured in `.goreleaser.yaml` under `brews` targeting `Lake-of-Tears/homebrew-tap` in directory `Formula`.
- **Credentials**: Uses repository secret `TAP_GITHUB_TOKEN` with write access to `Lake-of-Tears/homebrew-tap`.
- **Shell Completions**: GoReleaser `extra_install` injects `generate_completions_from_executable(bin/"angler", "completion")` into the generated formula.

---

## Phase 2: Upstream Homebrew Core Roadmap (`homebrew/core`)

Homebrew Core strictly requires that formulae are built from source rather than binary distributions, and requires tools to meet community notability guidelines.

### Source-Build Formula Specification

```ruby
class Angler < Formula
  desc "Official CLI for Lake of Tears — cast queries, launch clusters, and reel in data"
  homepage "https://thelakeoftears.com"
  url "https://github.com/Lake-of-Tears/angler/archive/refs/tags/v0.2.3.tar.gz"
  sha256 "<SOURCE_ARCHIVE_SHA256>"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X github.com/Lake-of-Tears/angler/cmd.Version=#{version}
      -X github.com/Lake-of-Tears/angler/cmd.BuiltBy=homebrew
    ]
    system "go", "build", *std_go_args(ldflags:), "./main.go"

    generate_completions_from_executable(bin/"angler", "completion")
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/angler version")
  end
end
```

### Submission Gate Checklist

| Requirement | Criteria | Current Status | Action |
|---|---|:---:|---|
| **Tagged Stable Releases** | Non-prerelease semantic releases published | ✅ Met | `v0.2.3` and later available |
| **Source Build** | Compiles cleanly with standard Go build flags | ✅ Met | `go build ./main.go` validated |
| **Shell Completions** | Generates completions dynamically from executable | ✅ Met | Built into `angler completion` |
| **Formula Audit & Test** | Passes `brew audit --new --strict` and `brew test` | ✅ Met | Audited against Homebrew guidelines |
| **Community Notability** | >= 150 GitHub stars, 75 forks, or 30 watchers | ⏳ Tracking | Monitor Lake-of-Tears/angler metrics |
| **Upstream PR** | Open pull request against `Homebrew/homebrew-core` | ⏳ Pending Gate | Submit once notability threshold reached |
