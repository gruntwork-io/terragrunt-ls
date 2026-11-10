/* --------------------------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for license information.
 * ------------------------------------------------------------------------------------------ */

import { workspace, ExtensionContext } from 'vscode';
import * as vscode from "vscode";
import * as path from 'path';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: ExtensionContext) {
	const isDevMode = context.extensionMode === vscode.ExtensionMode.Development;

	const serverOptions: ServerOptions = {
		run: {
			command: isDevMode
				? context.asAbsolutePath(path.join("node_modules", ".bin", "nodemon"))
				: context.asAbsolutePath(path.join('out', 'terragrunt-ls')),
			args: isDevMode ? [
				"-q",
				"--watch", "./**/*.go",
				"--signal", "SIGTERM",
				"--exec", "go", "run", "./main.go"
			] : allowArgs(),
			transport: TransportKind.stdio,
			options: isDevMode ? {
				cwd: context.asAbsolutePath(".."),
				env: { ...process.env, TG_LS_LOG: "debug.log" }
			} : undefined
		},
		debug: {
			command: context.asAbsolutePath(path.join("node_modules", ".bin", "nodemon")),
			args: [
				"-q",
				"--watch", "./**/*.go",
				"--signal", "SIGTERM",
				"--exec", "go", "run", "./main.go"
			],
			transport: TransportKind.stdio,
			options: {
				cwd: context.asAbsolutePath(".."),
				env: { ...process.env, TG_LS_LOG: "debug.log" }
			}
		}
	};

	// Options to control the language client
	const clientOptions: LanguageClientOptions = {
		// Register the server for Terragrunt files
		documentSelector: [
			{ scheme: 'file', language: 'terragrunt' },
		],
		synchronize: {
			// Notify the server about file changes to Terragrunt files
			fileEvents: workspace.createFileSystemWatcher('**/*.hcl')
		}
	};

	// Create the language client and start the client.
	client = new LanguageClient(
		'terragrunt-ls',
		'Terragrunt Language Server',
		serverOptions,
		clientOptions
	);

	// Start the client. This will also launch the server
	client.start();

	// The server reads its flags once, when it starts.
	context.subscriptions.push(workspace.onDidChangeConfiguration(async (event) => {
		if (!event.affectsConfiguration('terragrunt-ls.allow') && !event.affectsConfiguration('terragrunt-ls.allowCmds')) {
			return;
		}

		const choice = await vscode.window.showInformationMessage(
			'Reload the window to apply the new Terragrunt language server permissions.',
			'Reload Window'
		);
		if (choice === 'Reload Window') {
			await vscode.commands.executeCommand('workbench.action.reloadWindow');
		}
	}));
}

// allowArgs returns the server flags for the terragrunt-ls.allow and
// terragrunt-ls.allowCmds settings. Both settings have machine scope, so a
// repository's .vscode/settings.json cannot set them.
function allowArgs(): string[] {
	const config = workspace.getConfiguration('terragrunt-ls');
	const allow = config.get<string[]>('allow', []);
	const allowCmds = config.get<string[]>('allowCmds', []);

	return [
		...allow.map((capability) => `--allow=${capability}`),
		...allowCmds.flatMap((pattern) => ['--allow-cmd', pattern]),
	];
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}
