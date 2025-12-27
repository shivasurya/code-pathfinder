//@ts-check

'use strict';

const path = require('path');
const sveltePreprocess = require('svelte-preprocess');

/** @type {import('webpack').Configuration} */
const webviewConfig = (env, argv) => ({
  name: 'webview',
  target: 'web',
  mode: argv.mode || 'none',
  entry: './src/ui/svelte-webview/main.ts',
  output: {
    path: path.resolve(__dirname, 'dist', 'svelte-webview'),
    filename: 'webview.js',
    clean: false
  },
  resolve: {
    alias: {
      svelte: path.resolve(__dirname, 'node_modules', 'svelte', 'src', 'runtime'),
      'svelte/internal': path.resolve(__dirname, 'node_modules', 'svelte', 'src', 'runtime', 'internal', 'index.js'),
      'svelte/store': path.resolve(__dirname, 'node_modules', 'svelte', 'src', 'runtime', 'store', 'index.js'),
      'svelte/internal/disclose-version': path.resolve(__dirname, 'node_modules', 'svelte', 'src', 'runtime', 'internal', 'disclose-version', 'index.js')
    },
    extensions: ['.mjs', '.js', '.ts', '.svelte'],
    mainFields: ['svelte', 'browser', 'module', 'main'],
    conditionNames: ['svelte', 'browser', 'import', 'default']
  },
  module: {
    rules: [
      {
        test: /\.svelte$/,
        use: {
          loader: 'svelte-loader',
          options: {
            compilerOptions: {
              dev: argv.mode !== 'production'
            },
            emitCss: false,
            hotReload: argv.mode !== 'production',
            preprocess: sveltePreprocess()
          }
        }
      },
      {
        test: /\.ts$/,
        exclude: /node_modules/,
        use: 'ts-loader'
      },
      {
        test: /\.css$/,
        use: ['style-loader', 'css-loader']
      },
      {
        // Required to prevent errors from Svelte on Webpack 5+
        test: /node_modules\/svelte\/.*\.mjs$/,
        resolve: {
          fullySpecified: false
        }
      }
    ]
  },
  devtool: argv.mode === 'production' ? false : 'source-map',
  plugins: [
    new (require('copy-webpack-plugin'))({
      patterns: [
        { from: 'src/ui/svelte-webview/index.html', to: 'index.html' }
      ]
    })
  ]
});

module.exports = webviewConfig;
