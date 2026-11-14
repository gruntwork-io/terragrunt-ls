import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import { runTests } from '@vscode/test-electron';

// Downloads VS Code, opens testFixture with this extension loaded, and runs e2e.ts inside it.
// VSCODE_VERSION picks the VS Code release, such as 1.98.0 for the minimum in engines.vscode.
async function main(): Promise<void> {
	const root = path.resolve(__dirname, '..', '..');

	// VS Code creates its IPC socket in the user data dir. Under a deep checkout, the default
	// .vscode-test/user-data pushes the socket path past the 107-character Unix limit.
	const userDataDir = fs.mkdtempSync(path.join(os.tmpdir(), 'tg-ls-vscode-'));
	try {
		await runTests({
			version: process.env.VSCODE_VERSION ?? 'stable',
			extensionDevelopmentPath: root,
			extensionTestsPath: path.join(__dirname, 'e2e'),
			launchArgs: [
				path.join(root, 'testFixture'),
				'--disable-extensions',
				`--user-data-dir=${userDataDir}`,
			],
		});
	} finally {
		fs.rmSync(userDataDir, { recursive: true, force: true });
	}
}

main().catch((err) => {
	console.error(err);
	process.exit(1);
});
