# hayate

[日本語](README.ja.md)

hayate is a Go CLI that searches project directories across Windows 11 and WSL2. Select a project to jump to its directory and optionally run a configured action.

## Installation

Prebuilt Windows and Linux binaries are available from [GitHub Releases](https://github.com/withnic/hayate/releases) once a release is published.

With Go 1.23 or later, run this from any directory.

```sh
go install github.com/withnic/hayate/cmd/hayate@latest
```

If you already have a local clone, run `go install ./cmd/hayate` from the repository root to install that checkout.

Add the installation directory to your PATH. This is `GOBIN` if set, or the `bin` directory under `go env GOPATH`.

- From Windows PowerShell, the Windows build can search both Windows and WSL2 directories. Installing hayate inside WSL is optional.
- To run hayate directly from Bash or Zsh inside WSL, install it there and add it to your WSL PATH.

Windows 11 is the supported Windows version. Windows 10 has not been tested.

## Configuration

Run `hayate init-config` to create `~/.hayate.jsonc`, then edit `roots` to match your directory layout.

```jsonc
{
  "roots": [
    {"env": "windows", "path": "C:\\dev", "mode": "children"},
    {"env": "wsl", "distro": "Ubuntu", "path": "/home/alice/projects", "mode": "children"},
  ],
  "actions": {
    "c": {"description": "Open VS Code", "windows": "code .", "wsl": "code ."},
  },
  // Optional: uncomment to open VS Code when you run j without an action flag.
  // "default_action": "c",
}
```

### Set up a WSL search root

1. Open a WSL terminal and change to the parent directory containing the projects you want to search.
2. Run `pwd` there and copy the resulting absolute path into `path` for the WSL entry in `roots`. For example, if `pwd` prints `/home/alice/projects`, use this entry.

   ```jsonc
   {"env": "wsl", "distro": "Ubuntu", "path": "/home/alice/projects", "mode": "children"}
   ```

3. Set `distro` to the name of your distribution. Run `wsl -l -q` in PowerShell to check the name. After saving the configuration, run `hayate.exe list --refresh` in PowerShell to see the results.

## Set up and run `j`

### PowerShell

In PowerShell, run these two lines to register `j` for the current session and open the search.

```powershell
hayate.exe init powershell | Out-String | Invoke-Expression
j
```

To make `j` available in new PowerShell sessions, add the first line to your PowerShell profile (`$PROFILE`).

### Bash or Zsh in WSL

If you installed hayate inside WSL, run these lines from Bash. For Zsh, replace `bash` with `zsh` in the first line.

```bash
eval "$(hayate init bash)"
j
```

To make `j` available in new sessions, add the first line to `~/.bashrc` or `~/.zshrc` as appropriate.

## Usage

- Run `j` to open the search. Press Enter to jump to the selected directory, or Esc or Ctrl+C to cancel.
- Run `j -c` to open VS Code after changing directories. Add entries to `actions` to define more flags.
- Set `"default_action": "c"` at the top level of your config to open VS Code with plain `j`. An explicitly requested action takes priority. With `default_action` omitted, plain `j` only changes directories.
- If you have `withnic/knit` checked out as `/home/alice/projects/knit`, run `j --query knit` to open the search with `knit` already entered.
- Run `hayate list --json` to inspect the results. Add `--refresh` to rescan the roots.
- Pass `--cmd <name>` to `hayate init` to change the wrapper's name.

## Development checks

With GNU Make, run `make fmt` to format the Go source and `make check` to check formatting, run `go vet`, and test with the race detector. The race detector requires CGO and a C compiler.

To run checks directly, use `gofmt -s -l cmd internal` from the project root. If it lists any files, format them with `gofmt -s -w cmd internal`. Run `go vet ./...` for static analysis and `go test ./...` for the standard test suite.

## License

[MIT License](LICENSE)
