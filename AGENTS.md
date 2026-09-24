
Use README.md files to understand program intent and structure.

Binary outputs must build via `go run mage.go binary`.

Build without CGO.

Use the [afero package](github.com/spf13/afero) to abstract filesystem
access, ideally via the [pathlib package](github.com/chigopher/pathlib).

Use the [zap logger](go.uber.org/zap) for logging and favor using the
[logutil package](github.com/wrouesnel/go.logutil).

Replace the template `github.com/wrouesnel/golang-template` module path
with one for the actual project and then update this file once its done.

Unless there is good reason, implement all binary applications as
exportable packages under `pkg/entrypoints/<binary name>` and keep
code under `cmd/` to an absolute minimum.
