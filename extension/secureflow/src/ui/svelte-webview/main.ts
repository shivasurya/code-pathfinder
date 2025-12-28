import App from './App.svelte';

// Declare global vscode API
declare global {
  interface Window {
    acquireVsCodeApi(): any;
    modelConfig: any;
  }
}

const app = new App({
  target: document.body
});

export default app;
