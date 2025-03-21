import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

import sitemap from "@astrojs/sitemap";

// https://astro.build/config
export default defineConfig({
  site: 'https://codepathfinder.dev',
  integrations: [starlight({
    title: 'Code PathFinder',
    favicon: 'favicon.ico',
    social: {
      github: 'https://github.com/shivasurya/code-pathfinder',
      discord: 'https://discord.gg/xmPdJC6WPX'
    },
    sidebar: [{
      label: 'Getting Started',
      items: [
      // Each item here is one entry in the navigation menu.
      {
        label: 'Overview',
        slug: 'overview'
      }, {
        label: 'CLI Quickstart',
        slug: 'quickstart'
      }, {
        label: 'CLI Reference',
        slug: 'cli-reference'
      }, {
          label: 'CI Integration',
          slug: 'ci'
        }]
    }, {
      label: 'PathFinder Queries',
      autogenerate: {
        directory: 'queries'
      }
    }, {
      label: 'API Reference',
      autogenerate: {
        directory: 'api'
      }
    }, {
      label: 'Changelog',
      slug: 'changelog'
    },],
    customCss: ["./src/styles/font.css", "./src/styles/layout.css"],
    components: {
      Footer: './src/components/Footer.astro',
      Header: './src/components/Header.astro',
    },
  }), sitemap()]
});