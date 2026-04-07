import * as assert from 'assert';
import * as fs from 'fs';
import * as path from 'path';

const grammarPath = path.join(__dirname, '..', '..', 'syntaxes', 'terragrunt.tmGrammar.json');
const grammar = JSON.parse(fs.readFileSync(grammarPath, 'utf-8'));

// Helper: test a TextMate 'match' regex against a string
function matchPattern(pattern: string, input: string): RegExpMatchArray | null {
	return input.match(new RegExp(pattern));
}

describe('syntaxes/terragrunt.tmGrammar.json', () => {

	describe('top-level structure', () => {
		it('has the correct name', () => {
			assert.strictEqual(grammar.name, 'Terragrunt');
		});

		it('has scopeName "source.hcl.terragrunt"', () => {
			assert.strictEqual(grammar.scopeName, 'source.hcl.terragrunt');
		});

		it('includes #comments, #block, and #expressions in top-level patterns', () => {
			const includes = grammar.patterns.map((p: { include: string }) => p.include);
			assert.ok(includes.includes('#comments'));
			assert.ok(includes.includes('#block'));
			assert.ok(includes.includes('#expressions'));
		});

		it('has a repository with all expected rule names', () => {
			const expected = ['comments', 'block', 'expressions', 'strings', 'heredoc', 'numbers', 'booleans', 'functions', 'attribute', 'object_key'];
			for (const key of expected) {
				assert.ok(Object.prototype.hasOwnProperty.call(grammar.repository, key), `repository should have "${key}"`);
			}
		});
	});

	describe('comments', () => {
		const commentPatterns = grammar.repository.comments.patterns;

		it('has a line comment pattern for #', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.line.number-sign.hcl');
			assert.ok(p, 'number-sign line comment pattern should exist');
		});

		it('# comment pattern matches a hash comment', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.line.number-sign.hcl');
			assert.ok(matchPattern(p.match, '# this is a comment'));
			assert.ok(matchPattern(p.match, '#comment without space'));
		});

		it('# comment pattern does not match empty string', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.line.number-sign.hcl');
			assert.ok(!matchPattern(p.match, '  no hash here'));
		});

		it('has a line comment pattern for //', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.line.double-slash.hcl');
			assert.ok(p, 'double-slash line comment pattern should exist');
		});

		it('// comment pattern matches double-slash comment', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.line.double-slash.hcl');
			assert.ok(matchPattern(p.match, '// this is a comment'));
			assert.ok(matchPattern(p.match, '//no space'));
		});

		it('has a block comment pattern', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.block.hcl');
			assert.ok(p, 'block comment pattern should exist');
			assert.ok(p.begin);
			assert.ok(p.end);
		});

		it('block comment pattern begins with /* and ends with */', () => {
			const p = commentPatterns.find((c: { name: string }) => c.name === 'comment.block.hcl');
			assert.ok(matchPattern(p.begin, '/*'));
			assert.ok(matchPattern(p.end, '*/'));
		});
	});

	describe('block patterns', () => {
		const blockPatterns = grammar.repository.block.patterns;

		it('has two block patterns (keyword block and generic block)', () => {
			assert.strictEqual(blockPatterns.length, 2);
		});

		it('keyword block has name "meta.block.hcl"', () => {
			const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.hcl');
			assert.ok(p, 'meta.block.hcl pattern should exist');
		});

		it('generic block has name "meta.block.other.hcl"', () => {
			const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.other.hcl');
			assert.ok(p, 'meta.block.other.hcl pattern should exist');
		});

		describe('keyword block begin pattern', () => {
			let beginRegex: RegExp;

			before(() => {
				const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.hcl');
				beginRegex = new RegExp(p.begin);
			});

			const keywords = [
				'unit', 'stack', 'dependency', 'dependencies', 'include',
				'terraform', 'remote_state', 'locals', 'inputs', 'generate',
				'feature', 'exclude', 'errors', 'engine', 'catalog'
			];

			for (const kw of keywords) {
				it(`matches keyword "${kw}" block`, () => {
					assert.ok(beginRegex.test(`${kw} {`), `"${kw} {" should match block begin`);
				});
			}

			it('matches keyword block with a label string', () => {
				assert.ok(beginRegex.test('include "root" {'));
				assert.ok(beginRegex.test('dependency "vpc" {'));
				assert.ok(beginRegex.test('generate "provider" {'));
			});

			it('does not match a bare identifier that is not a keyword as meta.block.hcl', () => {
				assert.ok(!beginRegex.test('myblock {'));
			});
		});

		describe('generic block begin pattern', () => {
			let beginRegex: RegExp;

			before(() => {
				const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.other.hcl');
				beginRegex = new RegExp(p.begin);
			});

			it('matches any identifier block', () => {
				assert.ok(beginRegex.test('myblock {'));
				assert.ok(beginRegex.test('resource "aws_instance" {'));
			});

			it('does not match a line without {', () => {
				assert.ok(!beginRegex.test('myblock'));
			});
		});

		it('keyword block captures group 1 as entity.name.type.hcl', () => {
			const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.hcl');
			assert.strictEqual(p.beginCaptures['1'].name, 'entity.name.type.hcl');
		});

		it('keyword block captures group 2 as string.quoted.double.hcl', () => {
			const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.hcl');
			assert.strictEqual(p.beginCaptures['2'].name, 'string.quoted.double.hcl');
		});

		it('block end pattern matches closing brace', () => {
			const p = blockPatterns.find((b: { name: string }) => b.name === 'meta.block.hcl');
			assert.ok(new RegExp(p.end).test('}'));
		});

		it('blocks include nested comments, block, and expressions', () => {
			for (const p of blockPatterns) {
				const includes = p.patterns.map((pat: { include: string }) => pat.include);
				assert.ok(includes.includes('#comments'));
				assert.ok(includes.includes('#block'));
				assert.ok(includes.includes('#expressions'));
			}
		});
	});

	describe('expressions', () => {
		const exprPatterns = grammar.repository.expressions.patterns;

		it('includes all expected expression sub-patterns', () => {
			const includes = exprPatterns.map((p: { include: string }) => p.include);
			assert.ok(includes.includes('#strings'));
			assert.ok(includes.includes('#heredoc'));
			assert.ok(includes.includes('#numbers'));
			assert.ok(includes.includes('#booleans'));
			assert.ok(includes.includes('#functions'));
			assert.ok(includes.includes('#attribute'));
			assert.ok(includes.includes('#object_key'));
		});
	});

	describe('strings', () => {
		const strPattern = grammar.repository.strings.patterns[0];

		it('string pattern has name "string.quoted.double.hcl"', () => {
			assert.strictEqual(strPattern.name, 'string.quoted.double.hcl');
		});

		it('string begins and ends with double quote', () => {
			assert.strictEqual(strPattern.begin, '"');
			assert.strictEqual(strPattern.end, '"');
		});

		it('has escape sequence sub-pattern', () => {
			const escape = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'constant.character.escape.hcl'
			);
			assert.ok(escape, 'escape sequence pattern should exist');
		});

		it('escape pattern matches \\n, \\r, \\t, \\", \\\\', () => {
			const escape = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'constant.character.escape.hcl'
			);
			const re = new RegExp(escape.match);
			assert.ok(re.test('\\n'));
			assert.ok(re.test('\\r'));
			assert.ok(re.test('\\t'));
			assert.ok(re.test('\\"'));
			assert.ok(re.test('\\\\'));
		});

		it('escape pattern does not match non-escape sequences', () => {
			const escape = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'constant.character.escape.hcl'
			);
			const re = new RegExp(escape.match);
			assert.ok(!re.test('\\x'));
			assert.ok(!re.test('\\a'));
		});

		it('has string interpolation sub-pattern', () => {
			const interp = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'meta.interpolation.hcl'
			);
			assert.ok(interp, 'interpolation pattern should exist');
		});

		it('interpolation begins with ${ and ends with }', () => {
			const interp = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'meta.interpolation.hcl'
			);
			assert.ok(new RegExp(interp.begin).test('${'));
			assert.ok(new RegExp(interp.end).test('}'));
		});

		it('interpolation begin capture is punctuation.section.interpolation.begin.hcl', () => {
			const interp = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'meta.interpolation.hcl'
			);
			assert.strictEqual(interp.beginCaptures['0'].name, 'punctuation.section.interpolation.begin.hcl');
		});

		it('interpolation end capture is punctuation.section.interpolation.end.hcl', () => {
			const interp = strPattern.patterns.find(
				(p: { name: string }) => p.name === 'meta.interpolation.hcl'
			);
			assert.strictEqual(interp.endCaptures['0'].name, 'punctuation.section.interpolation.end.hcl');
		});
	});

	describe('heredoc', () => {
		const heredocPattern = grammar.repository.heredoc.patterns[0];

		it('heredoc pattern has name "string.unquoted.heredoc.hcl"', () => {
			assert.strictEqual(heredocPattern.name, 'string.unquoted.heredoc.hcl');
		});

		it('heredoc begin pattern matches <<IDENTIFIER', () => {
			const re = new RegExp(heredocPattern.begin);
			assert.ok(re.test('<<EOF'));
			assert.ok(re.test('<<HEREDOC'));
		});

		it('heredoc begin pattern matches <<-IDENTIFIER (indented heredoc)', () => {
			const re = new RegExp(heredocPattern.begin);
			assert.ok(re.test('<<-EOF'));
			assert.ok(re.test('<<-HEREDOC'));
		});

		it('heredoc begin captures identifier as keyword.control.heredoc.hcl', () => {
			assert.strictEqual(heredocPattern.beginCaptures['1'].name, 'keyword.control.heredoc.hcl');
		});
	});

	describe('numbers', () => {
		const numberPattern = grammar.repository.numbers.patterns[0];

		it('number pattern has name "constant.numeric.hcl"', () => {
			assert.strictEqual(numberPattern.name, 'constant.numeric.hcl');
		});

		it('matches integer numbers', () => {
			const re = new RegExp(numberPattern.match);
			assert.ok(re.test('42'));
			assert.ok(re.test('0'));
			assert.ok(re.test('100'));
		});

		it('matches floating-point numbers', () => {
			const re = new RegExp(numberPattern.match);
			assert.ok(re.test('3.14'));
			assert.ok(re.test('0.5'));
		});

		it('matches numbers with exponent', () => {
			const re = new RegExp(numberPattern.match);
			assert.ok(re.test('1e10'));
			assert.ok(re.test('2.5E-3'));
			assert.ok(re.test('1e+5'));
		});

		it('does not match a bare decimal point', () => {
			const re = new RegExp(numberPattern.match);
			assert.ok(!re.test('.5'));
		});
	});

	describe('booleans', () => {
		const boolPattern = grammar.repository.booleans.patterns[0];

		it('boolean pattern has name "constant.language.hcl"', () => {
			assert.strictEqual(boolPattern.name, 'constant.language.hcl');
		});

		it('matches true', () => {
			assert.ok(matchPattern(boolPattern.match, 'true'));
		});

		it('matches false', () => {
			assert.ok(matchPattern(boolPattern.match, 'false'));
		});

		it('matches null', () => {
			assert.ok(matchPattern(boolPattern.match, 'null'));
		});

		it('does not match partial words (word boundary enforcement)', () => {
			const re = new RegExp(boolPattern.match);
			assert.ok(!re.test('truevalue'));
			assert.ok(!re.test('notfalse'));
			assert.ok(!re.test('nullable'));
		});
	});

	describe('functions', () => {
		const funcPattern = grammar.repository.functions.patterns[0];

		it('function pattern has name "meta.function-call.hcl"', () => {
			assert.strictEqual(funcPattern.name, 'meta.function-call.hcl');
		});

		it('function capture group 1 has name "support.function.hcl"', () => {
			assert.strictEqual(funcPattern.captures['1'].name, 'support.function.hcl');
		});

		it('matches function calls', () => {
			const re = new RegExp(funcPattern.match);
			assert.ok(re.test('tostring('));
			assert.ok(re.test('read_terragrunt_config('));
			assert.ok(re.test('find_in_parent_folders('));
		});

		it('matches function calls with underscores and digits in name', () => {
			const re = new RegExp(funcPattern.match);
			assert.ok(re.test('func_1('));
			assert.ok(re.test('_privateFunc('));
		});

		it('does not match identifiers without opening parenthesis', () => {
			const re = new RegExp(funcPattern.match);
			assert.ok(!re.test('identifier'));
		});
	});

	describe('attribute', () => {
		const attrPattern = grammar.repository.attribute.patterns[0];

		it('attribute capture group 1 has name "variable.other.assignment.hcl"', () => {
			assert.strictEqual(attrPattern.captures['1'].name, 'variable.other.assignment.hcl');
		});

		it('matches attribute assignments', () => {
			const re = new RegExp(attrPattern.match);
			assert.ok(re.test('source = '));
			assert.ok(re.test('enabled = '));
			assert.ok(re.test('my_var = '));
		});

		it('does not match == (equality operator)', () => {
			const re = new RegExp(attrPattern.match);
			assert.ok(!re.test('x == y'), 'equality operator should not be matched as attribute assignment');
		});

		it('does not match identifiers without =', () => {
			const re = new RegExp(attrPattern.match);
			assert.ok(!re.test('source'));
		});
	});

	describe('object_key', () => {
		const objKeyPattern = grammar.repository.object_key.patterns[0];

		it('capture group 1 has name "support.type.hcl"', () => {
			assert.strictEqual(objKeyPattern.captures['1'].name, 'support.type.hcl');
		});

		it('capture group 2 has name "variable.other.member.hcl"', () => {
			assert.strictEqual(objKeyPattern.captures['2'].name, 'variable.other.member.hcl');
		});

		it('matches local.<name> references', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(re.test('local.my_var'));
			assert.ok(re.test('local.region'));
		});

		it('matches dependency.<name> references', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(re.test('dependency.vpc'));
			assert.ok(re.test('dependency.my_dep'));
		});

		it('matches include.<name> references', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(re.test('include.root'));
		});

		it('matches each.<name> references', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(re.test('each.key'));
			assert.ok(re.test('each.value'));
		});

		it('matches var.<name> references', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(re.test('var.region'));
		});

		it('matches module.<name> references', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(re.test('module.vpc'));
		});

		it('does not match unknown prefixes', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(!re.test('unknown.value'));
		});

		it('does not match a reference without a dot', () => {
			const re = new RegExp(objKeyPattern.match);
			assert.ok(!re.test('local'));
		});
	});
});