import { mount } from 'svelte';
import App from './App.svelte';

// Declare global vscode API
declare global {
  interface Window {
    acquireVsCodeApi(): any;
    modelConfig: any;
  }
}

const app = mount(App, {
  target: document.body
});

export default app;
