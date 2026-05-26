<script lang="ts">
  export let vscode: any;

  type Connection = 'github' | 'google-oauth2' | 'windowslive';

  function signIn(connection: Connection) {
    vscode?.postMessage({ type: 'auth:loginWithProvider', connection });
  }

  function continueAsGuest() {
    vscode?.postMessage({ type: 'auth:continueAsGuest' });
  }
</script>

<div class="login-page">
  <div class="brand">
    <svg
      class="logo"
      viewBox="47 67 400 400"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
    >
      <path
        d="M798 1560 318 -224H20L500 1560Z"
        fill="currentColor"
        transform="translate(106.399 339.076) scale(0.11151 -0.12266)"
      />
      <path
        d="M115 -418V1118H460V926H472Q493 976 533.0 1023.5Q573 1071 636.0 1101.5Q699 1132 788 1132Q906 1132 1007.5 1070.0Q1109 1008 1171.5 881.0Q1234 754 1234 558Q1234 368 1173.5 240.5Q1113 113 1011.5 48.5Q910 -16 786 -16Q658 -16 581.5 44.5Q505 105 472 181H464V-418ZM667 256Q768 256 821.5 339.0Q875 422 875 558Q875 694 821.5 776.5Q768 859 667 859Q567 859 511.5 778.0Q456 697 456 558Q456 421 511.5 338.5Q567 256 667 256Z"
        fill="currentColor"
        transform="translate(188.592 339.076) scale(0.11151 -0.11151)"
      />
      <path
        d="M361 -21Q281 -21 226.5 32.5Q172 86 172 165Q172 245 226.5 298.5Q281 352 361 352Q441 352 495.5 298.5Q550 245 550 165Q550 86 495.5 32.5Q441 -21 361 -21Z"
        fill="currentColor"
        transform="translate(325.090 339.076) scale(0.11151 -0.11151)"
      />
    </svg>
    <h1>Code Pathfinder</h1>
    <p class="tagline">
      AI-powered security analysis that finds vulnerabilities, models threats,
      and hardens your code — right in your editor.
    </p>
  </div>

  <div class="providers">
    <button
      class="provider sign-in google"
      type="button"
      on:click={() => signIn('google-oauth2')}
    >
      <span class="icon-chip">
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
          <path
            fill="#4285F4"
            d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
          />
          <path
            fill="#34A853"
            d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
          />
          <path
            fill="#FBBC05"
            d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
          />
          <path
            fill="#EA4335"
            d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
          />
        </svg>
      </span>
      <span class="label">Sign in with Google</span>
    </button>

    <button
      class="provider sign-in github"
      type="button"
      on:click={() => signIn('github')}
    >
      <span class="icon-chip">
        <svg
          viewBox="0 0 24 24"
          width="18"
          height="18"
          fill="#24292F"
          aria-hidden="true"
        >
          <path
            d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.55v-2.18c-3.2.7-3.88-1.34-3.88-1.34-.52-1.32-1.28-1.67-1.28-1.67-1.04-.71.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.76 2.7 1.25 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.23-1.28-5.23-5.69 0-1.26.45-2.29 1.18-3.1-.12-.29-.51-1.45.11-3.03 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 2.89-.39c.98 0 1.97.13 2.89.39 2.21-1.5 3.17-1.18 3.17-1.18.63 1.58.23 2.74.12 3.03.74.81 1.18 1.84 1.18 3.1 0 4.42-2.69 5.4-5.25 5.68.41.36.78 1.06.78 2.14v3.17c0 .31.21.67.8.55C20.21 21.39 23.5 17.08 23.5 12 23.5 5.65 18.35.5 12 .5z"
          />
        </svg>
      </span>
      <span class="label">Sign in with GitHub</span>
    </button>

    <!--
      Microsoft sign-in temporarily hidden until the Auth0 windowslive
      social connection is configured. Re-enable by uncommenting.
    <button
      class="provider sign-in microsoft"
      type="button"
      on:click={() => signIn('windowslive')}
    >
      <span class="icon-chip">
        <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden="true">
          <path fill="#F25022" d="M1 1h10v10H1z" />
          <path fill="#7FBA00" d="M13 1h10v10H13z" />
          <path fill="#00A4EF" d="M1 13h10v10H1z" />
          <path fill="#FFB900" d="M13 13h10v10H13z" />
        </svg>
      </span>
      <span class="label">Sign in with Microsoft</span>
    </button>
    -->

    <button class="provider guest" type="button" on:click={continueAsGuest}>
      <span class="icon-inline">
        <svg
          viewBox="0 0 24 24"
          width="18"
          height="18"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
          <circle cx="12" cy="7" r="4" />
        </svg>
      </span>
      <span class="label">Continue as Guest</span>
    </button>
  </div>
</div>

<style>
  .login-page {
    width: 100%;
    max-width: 360px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 32px;
    padding: 16px;
    box-sizing: border-box;
  }

  .brand {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: 12px;
  }

  .logo {
    width: 96px;
    height: 96px;
    color: var(--vscode-foreground);
  }

  h1 {
    margin: 0;
    font-size: 22px;
    font-weight: 600;
    color: var(--vscode-foreground);
  }

  .tagline {
    margin: 0;
    font-size: 13px;
    line-height: 1.5;
    color: var(--vscode-descriptionForeground);
    max-width: 320px;
  }

  .providers {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
  }

  .provider {
    display: flex;
    width: 100%;
    padding: 0;
    border: 1px solid transparent;
    border-radius: 6px;
    font-family: var(--vscode-font-family);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    overflow: hidden;
    box-sizing: border-box;
    transition:
      background-color 0.15s,
      transform 0.1s;
  }

  .provider:hover {
    transform: translateY(-1px);
  }

  .provider:active {
    transform: translateY(0);
  }

  .label {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    text-align: center;
  }

  /* Sign-in buttons: white icon chip on the left, label centered */
  .sign-in {
    align-items: stretch;
  }

  .sign-in .icon-chip {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    background: #ffffff;
    flex-shrink: 0;
  }

  .sign-in .label {
    padding: 10px 12px;
  }

  /* Color variants */
  .google {
    background: #4285f4;
    color: #ffffff;
  }
  .google:hover {
    background: #3367d6;
  }

  .github {
    background: #24292f;
    color: #ffffff;
  }
  .github:hover {
    background: #1c2128;
  }

  .microsoft {
    background: #ffffff;
    color: #1f1f1f;
    border-color: rgba(0, 0, 0, 0.12);
  }
  .microsoft:hover {
    background: #f3f3f3;
  }

  /* Guest: inline icon + text, both centered, no chip */
  .guest {
    align-items: center;
    justify-content: center;
    gap: 12px;
    padding: 10px 16px;
    background: transparent;
    color: var(--vscode-foreground);
    border-color: var(
      --vscode-button-border,
      var(--vscode-widget-border, rgba(128, 128, 128, 0.4))
    );
  }
  .guest:hover {
    background: var(--vscode-list-hoverBackground);
  }
  .guest .icon-inline {
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }
  .guest .label {
    flex: 0 0 auto;
  }
</style>
