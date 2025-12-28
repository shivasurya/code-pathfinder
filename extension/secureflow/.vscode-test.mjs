import { defineConfig } from '@vscode/test-cli';

export default defineConfig({
	files: 'out/test/**/*.test.js',
	coverage: {
		includeAll: true,
		exclude: ['**/test/**', '**/node_modules/**', '**/out/**'],
		reporter: ['text', 'html', 'lcov']
	}
});
