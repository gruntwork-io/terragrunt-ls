import * as assert from 'assert';
import * as fs from 'fs';
import * as path from 'path';

const configPath = path.join(__dirname, '..', '..', 'language-configuration.json');
const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'));

describe('language-configuration.json', () => {

	describe('comments', () => {
		it('defines line comment as #', () => {
			assert.strictEqual(config.comments.lineComment, '#');
		});

		it('defines block comment as /* */', () => {
			assert.deepStrictEqual(config.comments.blockComment, ['/*', '*/']);
		});
	});

	describe('brackets', () => {
		it('includes curly braces as bracket pair', () => {
			assert.ok(config.brackets.some((b: string[]) => b[0] === '{' && b[1] === '}'));
		});

		it('includes square brackets as bracket pair', () => {
			assert.ok(config.brackets.some((b: string[]) => b[0] === '[' && b[1] === ']'));
		});

		it('includes parentheses as bracket pair', () => {
			assert.ok(config.brackets.some((b: string[]) => b[0] === '(' && b[1] === ')'));
		});

		it('has exactly 3 bracket pairs', () => {
			assert.strictEqual(config.brackets.length, 3);
		});
	});

	describe('autoClosingPairs', () => {
		it('auto-closes curly braces', () => {
			assert.ok(config.autoClosingPairs.some(
				(p: { open: string; close: string }) => p.open === '{' && p.close === '}'
			));
		});

		it('auto-closes square brackets', () => {
			assert.ok(config.autoClosingPairs.some(
				(p: { open: string; close: string }) => p.open === '[' && p.close === ']'
			));
		});

		it('auto-closes parentheses', () => {
			assert.ok(config.autoClosingPairs.some(
				(p: { open: string; close: string }) => p.open === '(' && p.close === ')'
			));
		});

		it('auto-closes double quotes but not inside strings', () => {
			const quotePair = config.autoClosingPairs.find(
				(p: { open: string; close: string; notIn?: string[] }) =>
					p.open === '"' && p.close === '"'
			);
			assert.ok(quotePair, 'double-quote auto-closing pair should exist');
			assert.deepStrictEqual(quotePair.notIn, ['string']);
		});
	});

	describe('surroundingPairs', () => {
		it('includes curly braces as surrounding pair', () => {
			assert.ok(config.surroundingPairs.some((p: string[]) => p[0] === '{' && p[1] === '}'));
		});

		it('includes square brackets as surrounding pair', () => {
			assert.ok(config.surroundingPairs.some((p: string[]) => p[0] === '[' && p[1] === ']'));
		});

		it('includes parentheses as surrounding pair', () => {
			assert.ok(config.surroundingPairs.some((p: string[]) => p[0] === '(' && p[1] === ')'));
		});

		it('includes double quotes as surrounding pair', () => {
			assert.ok(config.surroundingPairs.some((p: string[]) => p[0] === '"' && p[1] === '"'));
		});

		it('has exactly 4 surrounding pairs', () => {
			assert.strictEqual(config.surroundingPairs.length, 4);
		});
	});

	describe('indentationRules', () => {
		let increasePattern: RegExp;
		let decreasePattern: RegExp;

		before(() => {
			increasePattern = new RegExp(config.indentationRules.increaseIndentPattern);
			decreasePattern = new RegExp(config.indentationRules.decreaseIndentPattern);
		});

		it('has increaseIndentPattern defined', () => {
			assert.ok(config.indentationRules.increaseIndentPattern);
		});

		it('has decreaseIndentPattern defined', () => {
			assert.ok(config.indentationRules.decreaseIndentPattern);
		});

		it('increaseIndentPattern matches terragrunt block keyword followed by {', () => {
			assert.ok(increasePattern.test('terraform {'));
			assert.ok(increasePattern.test('locals {'));
			assert.ok(increasePattern.test('unit {'));
			assert.ok(increasePattern.test('stack {'));
			assert.ok(increasePattern.test('dependency "dep_name" {'));
			assert.ok(increasePattern.test('include "root" {'));
			assert.ok(increasePattern.test('remote_state {'));
			assert.ok(increasePattern.test('generate "provider" {'));
			assert.ok(increasePattern.test('feature "my_feature" {'));
			assert.ok(increasePattern.test('catalog {'));
			assert.ok(increasePattern.test('engine {'));
			assert.ok(increasePattern.test('errors {'));
			assert.ok(increasePattern.test('exclude {'));
		});

		it('increaseIndentPattern matches assignment block { on same line', () => {
			assert.ok(increasePattern.test('  inputs = {'));
			assert.ok(increasePattern.test('  config = {'));
		});

		it('increaseIndentPattern matches bare { on a line', () => {
			assert.ok(increasePattern.test('  {'));
			assert.ok(increasePattern.test('{'));
		});

		it('increaseIndentPattern does not match a closing brace line', () => {
			assert.ok(!increasePattern.test('}'));
		});

		it('decreaseIndentPattern matches closing brace', () => {
			assert.ok(decreasePattern.test('}'));
			assert.ok(decreasePattern.test('  }'));
		});

		it('decreaseIndentPattern does not match an opening brace', () => {
			assert.ok(!decreasePattern.test('{'));
		});

		it('increaseIndentPattern matches HCL root-level keywords: resource, data, variable, output, module, provider', () => {
			assert.ok(increasePattern.test('resource {'));
			assert.ok(increasePattern.test('data {'));
			assert.ok(increasePattern.test('variable {'));
			assert.ok(increasePattern.test('output {'));
			assert.ok(increasePattern.test('module {'));
			assert.ok(increasePattern.test('provider {'));
		});

		it('increaseIndentPattern does not match keyword without opening brace', () => {
			assert.ok(!increasePattern.test('terraform'));
			assert.ok(!increasePattern.test('locals'));
		});
	});
});