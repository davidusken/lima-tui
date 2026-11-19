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
3. Run these before pushing:
   ```bash
   make fmt
   make vet
   make build
   ```
4. Try your change against real Lima VMs (start/stop/connect, both themes if you touched the UI).
5. Update `README.md` if you change key bindings, flags or behaviour.
6. Open the PR with a short description of what changed and why. Screenshots help for UI changes.

CI runs `go vet` and `go build` on every PR.

### Commit messages

Short, imperative summaries are preferred, optionally with a prefix:
`fix: handle VMs with no IP`, `feat: add filter by status`, `docs: …`, `chore: …`.
Commits starting with `docs:`, `chore:` or `test:` are left out of release notes.

## Releases

Releases are cut by the maintainer by pushing a `vX.Y.Z` tag; GoReleaser builds the
binaries and updates the Homebrew tap. You don't need to bump any version in your PR.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE).
