import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://astro.build/config
export default defineConfig({
	integrations: [
		starlight({
			title: 'Code PathFinder',
			social: {
				github: 'https://github.com/shivasurya/code-pathfinder',
				discord: 'https://discord.gg/xmPdJC6WPX',
			},
			sidebar: [
				{
					label: 'Getting Started',
					items: [
						// Each item here is one entry in the navigation menu.
						{ label: 'Overview', slug: 'overview' },
						{ label: 'CLI Quickstart', slug: 'quickstart' },
						{ label: 'CLI Reference', slug: 'cli-reference' },
					],
				},
				{
					label: 'PathFinder Queries',
					autogenerate: { directory: 'queries' },
				},
				{
					label: 'API Reference',
					autogenerate: { directory: 'api' },
				},
			],
		}),
	],
});
