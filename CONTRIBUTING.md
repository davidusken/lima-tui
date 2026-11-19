# Contributing to lima-tui

Thanks for your interest! Bug reports, ideas and pull requests are all welcome.

## Reporting bugs and requesting features

Open an [issue](https://github.com/davidusken/lima-tui/issues). For bugs, please include:

- `lima-tui --version`
- `limactl --version` and your OS (macOS/Linux, Intel/Apple Silicon)
- What you did, what you expected, and what happened instead
- Output of `limactl list` if the problem is about how VMs are shown

For bigger changes, open an issue first so we can agree on the approach before you spend time on it.

## Development setup

You need Go (see the version in `go.mod`) and a working [Lima](https://lima-vm.io/) install.

```bash
git clone https://github.com/<you>/lima-tui.git
cd lima-tui
make run      # run from source
make build    # build ./lima-tui
```

## Making a pull request

1. Fork the repo and create a branch from `main`.
2. Keep the change focused: one fix or feature per PR.
3. Run the checks before pushing:
   ```bash
   make check   # gofmt, go vet, tests (with -race), build
   make lint    # golangci-lint, if you have it installed
   ```
4. Try your change against real Lima VMs (start/stop/connect, both built-in themes and a skin with `bg: default` if you touched the UI).
5. Key bindings live in one table, `bindings` in `keys.go`; the key handler, help
   panel, header and status bar are built from it. Also update the table in
   `README.md` (`TestReadmeListsEveryKey` fails if a key is missing there).
6. Open the PR with a short description of what changed and why. Screenshots help for UI changes.

CI checks gofmt and runs vet, tests and golangci-lint on macOS and Linux for every PR.

### Commit messages

Short, imperative summaries are preferred, optionally with a prefix:
`fix: handle VMs with no IP`, `feat: add filter by status`, `docs: …`, `chore: …`.
Commits starting with `docs:`, `chore:` or `test:` are left out of release notes.

## Code layout

| File | What's in it |
|---|---|
| `main.go` | Flags and startup |
| `app.go` | App state, background refresh, status bar |
| `keys.go` | The key binding table and the key handler |
| `actions.go` | Connect, start/stop, restart, delete |
| `ui.go` | Layout, table rendering, help panel |
| `header.go` | Info block and key hints at the top |
| `theme.go` | The `theme` struct, built-in dark/light themes, applying a theme |
| `skin.go` | Parsing YAML skins and loading the theme list |
| `config.go` | Config folder, `config.yaml`, and resolving `--skin` / `skin:` |
| `vm.go` | Talking to `limactl` and formatting its output |
| `assets/` | README screenshots and demo GIFs; `assets/demo/` records them |

All `limactl` calls run off the UI goroutine; UI state is only changed inside
`QueueUpdateDraw` or key handlers.

Tests never run the real `limactl`: the app calls it through its `listVMs` and
`runCommand` fields, which `newTestApp` replaces with fakes. For code that
reports back through `QueueUpdateDraw`, `loop_test.go` runs the real event loop
on a simulated screen (`startLoop`, `onLoop`, `eventually`).

## Screenshots and demo GIFs

If you change how the UI looks, re-record the README images with `make demo`
(needs [VHS](https://github.com/charmbracelet/vhs): `brew install vhs`). It runs
`assets/demo/demo.tape.in` once per colour scheme against a fake `limactl`
(`assets/demo/limactl`) with made-up VMs, so your own VMs are neither touched
nor shown.

## Adding a colour or a skin

- A new themeable colour goes in three places: the `theme` struct and both
  built-ins in `theme.go`, `skinFile` plus the `fields` table in `parseSkin`
  (`skin.go`), and the skin key list in `README.md`.
- Example skins live in `skins/`. `TestExampleSkins` loads every one of them, so
  a typo fails `make check`. Name the palette's source in a comment at the top,
  and stick to palettes whose licence lets us ship them.

## Releases

The maintainer cuts releases by pushing a `vX.Y.Z` tag. GoReleaser then builds the
binaries, publishes the GitHub release and updates the Homebrew tap. Don't bump any
version in your PR.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
