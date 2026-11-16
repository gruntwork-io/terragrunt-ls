import * as assert from 'assert';
import * as path from 'path';
import * as vscode from 'vscode';

const fixture = path.resolve(__dirname, '..', '..', 'testFixture', 'terragrunt.hcl');

// The position of `greeting` in `local.greeting` on the fixture's inputs line.
const localReference = new vscode.Position(5, 21);

// Runs inside the VS Code that runTest.ts launches. VS Code fails the run when the promise rejects.
export async function run(): Promise<void> {
	const document = await vscode.workspace.openTextDocument(fixture);
	await vscode.window.showTextDocument(document);
	assert.strictEqual(document.languageId, 'terragrunt');

	const extension = vscode.extensions.getExtension('Gruntwork.terragrunt-ls');
	assert.ok(extension, 'extension Gruntwork.terragrunt-ls is not installed');
	await extension.activate();

	const hover = await waitForHover(document.uri, localReference, 30_000);
	assert.match(hover, /greeting = "hello"/);
}

// Polls for a hover, since the server answers only after it starts and parses the document.
async function waitForHover(uri: vscode.Uri, position: vscode.Position, timeoutMs: number): Promise<string> {
	const deadline = Date.now() + timeoutMs;
	while (Date.now() < deadline) {
		const hovers = await vscode.commands.executeCommand<vscode.Hover[]>('vscode.executeHoverProvider', uri, position);
		const text = hovers
			.flatMap((hover) => hover.contents)
			.map((content) => (typeof content === 'string' ? content : content.value))
			.join('\n');
		if (text !== '') {
			return text;
		}
		await new Promise((resolve) => setTimeout(resolve, 500));
	}
	throw new Error(`no hover at ${position.line}:${position.character} within ${timeoutMs}ms`);
}
