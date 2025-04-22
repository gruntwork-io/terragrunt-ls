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
			] : [],
			transport: TransportKind.stdio,
			options: isDevMode ? {
				cwd: context.asAbsolutePath(".."),
				env: isDevMode ? { ...process.env, TG_LS_LOG: "debug.log" } : process
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
		documentSelector: [{ scheme: 'file', language: 'hcl' }],
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
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}
