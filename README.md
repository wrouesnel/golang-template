# Golang Template repository

This is a template repository outlining the basic structure to be used for
Golang applications.

The content here should be replaced with a description of the program, instructions
for building (which should just be `go run mage.go binary`) as well as usage and
implementation details.

If this is a web application, include a screen shot of the main screen linked just
below the title.

## Template structure

| Path | Purpose |
|---|---|
| `cmd/<binary>/main.go` | Minimal `main` package for each binary. It only calls the entrypoint. |
| `pkg/entrypoints/<binary>/` | The real application: command line parsing, logging, config loading and commands. |
| `pkg/` | Other exportable packages of the application. |
| `version/` | Application name, description and version. `Version` is set at link time by the build. |
| `golang-template.yml` | Default configuration file, loaded from the working directory. |
| `magefile.go`, `mage.go` | The build system. Run `go run mage.go -l` to list targets. |
| `web/` | Optional web interface. It's built and embedded if `web/package.json` exists. |
| `Dockerfile` | Container image, built and pushed by `.github/workflows/container.yml`. |
| `.github/workflows/` | CI: build and test on every push, releases on `v*` tags. |

## Building

```sh
go run mage.go binary
./application-sample
```

`go run mage.go binary` builds into `bin/` and symlinks each binary into the repository root.
`go run mage.go releaseAll` cross-compiles release archives for every platform into
`release/`.

### Versions

The build stamps every binary with a version which sorts in the order builds were made
(Semantic Versioning 2.0, from git):

| The commit | Its version |
|---|---|
| tagged `vX.Y.Z` (a release) | `vX.Y.Z` |
| N commits after it, on the default branch or a detached HEAD | `vX.Y.(Z+1)-dev.N.gHASH` |
| before any tag | `v0.0.1-dev.N.gHASH`, N its commits in all |
| on another branch | `vX.Y.(Z+1)-branch-NAME.N.gHASH`, below the default branch's |
| with uncommitted changes | the above, ending `-dirty` |

Release by tagging `vX.Y.Z` on the default branch, which is never rewritten, so N only grows.
In a repository whose other tags are another program's, tag `PREFIXvX.Y.Z` and set
`VERSION_TAG_MATCH=PREFIXv*`. `VERSION` and `VERSION_SHORT` override it. CI needs the whole
history (`fetch-depth: 0`) for N.

### Link flags

`EXTRA_LDFLAGS` adds to the binaries' link flags, as `-X` symbols a build sets beside the version
(`EXTRA_LDFLAGS="-X example.com/app/pkg/x.Value=..." go run mage.go binary`). They're recorded
beside the binaries (`bin/*/.extra-ldflags`), so changing them alone builds the binaries again.

## Configuration

The binary loads `golang-template.yml` from the working directory, or the file named by
`--config-file`, and fails if it doesn't exist. Its keys map to `EntrypointConfig` in
`pkg/entrypoints/application_sample/config.go`.
