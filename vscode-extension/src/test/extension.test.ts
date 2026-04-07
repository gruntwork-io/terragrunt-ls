/**
 * Tests for the documentSelector change in src/extension.ts.
 *
 * The activate() function in extension.ts requires the VS Code runtime
 * (vscode module, LanguageClient, workspace API), so we cannot invoke it
 * directly in a plain Node.js/Mocha environment. Instead, these tests
 * validate the static configuration expressed in the source by reading
 * the TypeScript source file directly, ensuring the PR intent is captured:
 * the documentSelector must include entries for both 'terragrunt' and 'hcl'.
 */
import * as assert from 'assert';
import * as fs from 'fs';
import * as path from 'path';

const extensionSourcePath = path.join(__dirname, '..', '..', 'src', 'extension.ts');
const extensionSource = fs.readFileSync(extensionSourcePath, 'utf-8');

describe('src/extension.ts – documentSelector', () => {

	it('source file exists', () => {
		assert.ok(fs.existsSync(extensionSourcePath), 'extension.ts should exist');
	});

	it('documentSelector includes terragrunt language entry', () => {
		// Verify the selector array contains a terragrunt entry
		assert.ok(
			extensionSource.includes("language: 'terragrunt'"),
			"documentSelector should include { language: 'terragrunt' }"
		);
	});

	it('documentSelector includes hcl language entry', () => {
		assert.ok(
			extensionSource.includes("language: 'hcl'"),
			"documentSelector should include { language: 'hcl' }"
		);
	});

	it('both documentSelector entries use the file scheme', () => {
		// Count occurrences of scheme: 'file' near documentSelector definition
		const selectorBlock = extractDocumentSelectorBlock(extensionSource);
		assert.ok(selectorBlock, 'should be able to locate the documentSelector block');
		const fileSchemeCount = (selectorBlock.match(/scheme:\s*'file'/g) || []).length;
		assert.ok(fileSchemeCount >= 2, `Expected at least 2 'scheme: file' entries in documentSelector, found ${fileSchemeCount}`);
	});

	it('terragrunt entry appears before hcl entry in documentSelector (terragrunt takes precedence)', () => {
		const terragruntPos = extensionSource.indexOf("language: 'terragrunt'");
		const hclPos = extensionSource.indexOf("language: 'hcl'");
		assert.ok(terragruntPos !== -1, 'terragrunt entry should be present');
		assert.ok(hclPos !== -1, 'hcl entry should be present');
		assert.ok(
			terragruntPos < hclPos,
			'terragrunt selector entry should appear before hcl selector entry'
		);
	});

	it('exports an activate function', () => {
		assert.ok(
			extensionSource.includes('export function activate'),
			'extension.ts should export an activate function'
		);
	});

	it('exports a deactivate function', () => {
		assert.ok(
			extensionSource.includes('export function deactivate'),
			'extension.ts should export a deactivate function'
		);
	});

	it('uses workspace.createFileSystemWatcher for *.hcl files', () => {
		assert.ok(
			extensionSource.includes("createFileSystemWatcher('**/*.hcl')"),
			"should watch **/*.hcl files via createFileSystemWatcher"
		);
	});

	it('client id is "terragrunt-ls"', () => {
		assert.ok(
			extensionSource.includes("'terragrunt-ls'"),
			'LanguageClient should be created with id "terragrunt-ls"'
		);
	});
});

/**
 * Extracts the documentSelector array literal from extension.ts source text.
 * Returns the substring from 'documentSelector' up to the closing bracket.
 */
function extractDocumentSelectorBlock(source: string): string | null {
	const startIndex = source.indexOf('documentSelector:');
	if (startIndex === -1) { return null; }
	// Find the opening [ for the array
	const arrayStart = source.indexOf('[', startIndex);
	if (arrayStart === -1) { return null; }
	// Walk forward to find the matching ]
	let depth = 0;
	for (let i = arrayStart; i < source.length; i++) {
		if (source[i] === '[') { depth++; }
		else if (source[i] === ']') {
			depth--;
			if (depth === 0) {
				return source.substring(startIndex, i + 1);
			}
		}
	}
	return null;
}