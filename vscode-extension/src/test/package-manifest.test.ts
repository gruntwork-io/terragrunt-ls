import * as assert from 'assert';
import * as fs from 'fs';
import * as path from 'path';

const manifestPath = path.join(__dirname, '..', '..', 'package.json');
const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'));

describe('package.json contributes', () => {

	describe('languages contribution', () => {
		it('contributes at least one language', () => {
			assert.ok(Array.isArray(manifest.contributes.languages));
			assert.ok(manifest.contributes.languages.length > 0);
		});

		it('contributes a language with id "terragrunt"', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			assert.ok(lang, 'terragrunt language contribution should exist');
		});

		it('terragrunt language has aliases including "Terragrunt" and "terragrunt"', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			assert.ok(Array.isArray(lang.aliases));
			assert.ok(lang.aliases.includes('Terragrunt'));
			assert.ok(lang.aliases.includes('terragrunt'));
		});

		it('terragrunt language is associated with terragrunt.hcl filename', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			assert.ok(Array.isArray(lang.filenames));
			assert.ok(lang.filenames.includes('terragrunt.hcl'));
		});

		it('terragrunt language is associated with terragrunt.stack.hcl filename', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			assert.ok(lang.filenames.includes('terragrunt.stack.hcl'));
		});

		it('terragrunt language is associated with terragrunt.values.hcl filename', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			assert.ok(lang.filenames.includes('terragrunt.values.hcl'));
		});

		it('terragrunt language references language-configuration.json', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			assert.strictEqual(lang.configuration, './language-configuration.json');
		});

		it('language-configuration.json file exists at the referenced path', () => {
			const lang = manifest.contributes.languages.find(
				(l: { id: string }) => l.id === 'terragrunt'
			);
			const configFilePath = path.join(__dirname, '..', '..', lang.configuration);
			assert.ok(fs.existsSync(configFilePath), `language-configuration.json should exist at ${configFilePath}`);
		});
	});

	describe('grammars contribution', () => {
		it('contributes at least one grammar', () => {
			assert.ok(Array.isArray(manifest.contributes.grammars));
			assert.ok(manifest.contributes.grammars.length > 0);
		});

		it('contributes a grammar for the terragrunt language', () => {
			const grammar = manifest.contributes.grammars.find(
				(g: { language: string }) => g.language === 'terragrunt'
			);
			assert.ok(grammar, 'grammar for terragrunt language should exist');
		});

		it('terragrunt grammar has scopeName "source.hcl.terragrunt"', () => {
			const grammar = manifest.contributes.grammars.find(
				(g: { language: string }) => g.language === 'terragrunt'
			);
			assert.strictEqual(grammar.scopeName, 'source.hcl.terragrunt');
		});

		it('terragrunt grammar references syntaxes/terragrunt.tmGrammar.json', () => {
			const grammar = manifest.contributes.grammars.find(
				(g: { language: string }) => g.language === 'terragrunt'
			);
			assert.strictEqual(grammar.path, './syntaxes/terragrunt.tmGrammar.json');
		});

		it('grammar file exists at the referenced path', () => {
			const grammar = manifest.contributes.grammars.find(
				(g: { language: string }) => g.language === 'terragrunt'
			);
			const grammarFilePath = path.join(__dirname, '..', '..', grammar.path);
			assert.ok(fs.existsSync(grammarFilePath), `grammar file should exist at ${grammarFilePath}`);
		});
	});

	describe('consistency between languages and grammars', () => {
		it('every language has a corresponding grammar', () => {
			for (const lang of manifest.contributes.languages) {
				const grammar = manifest.contributes.grammars.find(
					(g: { language: string }) => g.language === lang.id
				);
				assert.ok(grammar, `language "${lang.id}" should have a corresponding grammar`);
			}
		});

		it('every grammar references an existing language id', () => {
			const langIds = manifest.contributes.languages.map((l: { id: string }) => l.id);
			for (const grammar of manifest.contributes.grammars) {
				assert.ok(
					langIds.includes(grammar.language),
					`grammar language "${grammar.language}" should match a defined language id`
				);
			}
		});
	});
});