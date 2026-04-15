# Developing the Visual Studio Code Extension

## Setup

- Read the [setup docs](../../docs/setup.md) for instructions on how to setup your local development environment.

## Running the Extension in Development Mode

- See the [setup docs](../../docs/setup.md) for instructions on how to setup your local development environment.
- Run `npm install` in this folder. This installs all necessary npm modules in both the client and server folder
- Open VS Code on this folder.
- Press Ctrl+Shift+B to start compiling the client and server in [watch mode](https://code.visualstudio.com/docs/editor/tasks#:~:text=The%20first%20entry%20executes,the%20HelloWorld.js%20file.).
- Switch to the Run and Debug View in the Sidebar (Ctrl+Shift+D).
- Select `Launch Client` from the drop down (if it is not already).
- Press ▷ to run the launch config (F5).
- In the [Extension Development Host](https://code.visualstudio.com/api/get-started/your-first-extension#:~:text=Then%2C%20inside%20the%20editor%2C%20press%20F5.%20This%20will%20compile%20and%20run%20the%20extension%20in%20a%20new%20Extension%20Development%20Host%20window.) instance of VSCode, open a Terragrunt HCL file.
- See the [capabilities documentation](../../docs/server-capabilities.md) for what the language server can do.

## Packaging

The VSIX is built by `@vscode/vsce`, which runs the `vscode:prepublish` script from `package.json` before it assembles the archive. Our prepublish pipeline does two things:

1. `npm run bundle` — runs `esbuild.js` with `--production`, which bundles `src/extension.ts` (and its `vscode-languageclient` dependency) into a single minified `out/extension.js`. The `vscode` module is marked external because it is provided by the VS Code runtime.
2. `./scripts/package.sh` — cross-compiles the `terragrunt-ls` Go binary into `out/terragrunt-ls` (or `out/terragrunt-ls.exe`) using the `GOOS`/`GOARCH` env vars passed by the release matrix.

`vsce` then zips everything not excluded by `.vscodeignore` into the `.vsix`. Only the bundled extension, the Go binary, and a handful of static assets (`package.json`, grammar, icon, README, LICENSE) end up in the package — no `node_modules`, no TypeScript sources.

### Commands

- `npm run bundle` — one-shot production bundle (used by `vscode:prepublish`).
- `npm run bundle:watch` — rebundle on change; useful if you want to test the bundled output locally.
- `npm run compile` / `npm run watch` — plain `tsc -b` build for local F5 debugging (the `.vscode/launch.json` pre-launch task runs `watch`). These do *not* bundle; they produce the same `out/extension.js` path but rely on `node_modules` at runtime, which is fine inside the Extension Development Host.

### Producing a VSIX locally

From `vscode-extension/`:

```sh
npx @vscode/vsce package --target <target>
```

where `<target>` is one of `linux-x64`, `linux-arm64`, `darwin-x64`, `darwin-arm64`, `win32-x64`, `win32-arm64` (these must match the `GOOS`/`GOARCH` combinations expected by `scripts/package.sh`). The release workflow at `.github/workflows/release.yml` runs the same command across all six targets.
