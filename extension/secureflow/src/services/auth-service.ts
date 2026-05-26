import * as vscode from 'vscode';
import * as crypto from 'crypto';
import {
  AUTH0_CLIENT_ID,
  AUTH0_CONNECTION,
  AUTH0_DOMAIN,
  AUTH0_REDIRECT_URI,
  AUTH0_SCOPES,
  AUTH_SECRET_KEYS
} from '../settings/auth-config';

export interface UserInfo {
  sub: string;
  name?: string;
  nickname?: string;
  email?: string;
  picture?: string;
}

interface PendingLogin {
  codeVerifier: string;
  state: string;
  resolve: (info: UserInfo) => void;
  reject: (err: Error) => void;
}

export class AuthService implements vscode.UriHandler {
  private static instance: AuthService;
  private context?: vscode.ExtensionContext;
  private pending?: PendingLogin;
  private readonly _onDidChangeSession = new vscode.EventEmitter<
    UserInfo | undefined
  >();
  public readonly onDidChangeSession = this._onDidChangeSession.event;

  private constructor() {}

  public static getInstance(): AuthService {
    if (!AuthService.instance) {
      AuthService.instance = new AuthService();
    }
    return AuthService.instance;
  }

  public initialize(context: vscode.ExtensionContext): void {
    this.context = context;
    context.subscriptions.push(vscode.window.registerUriHandler(this));
    context.subscriptions.push(this._onDidChangeSession);
  }

  public async getCurrentUser(): Promise<UserInfo | undefined> {
    const raw = await this.requireContext().secrets.get(
      AUTH_SECRET_KEYS.userInfo
    );
    return raw ? (JSON.parse(raw) as UserInfo) : undefined;
  }

  public async getAccessToken(): Promise<string | undefined> {
    return this.requireContext().secrets.get(AUTH_SECRET_KEYS.accessToken);
  }

  public async login(): Promise<UserInfo> {
    if (this.pending) {
      throw new Error('A login is already in progress.');
    }

    const codeVerifier = generateCodeVerifier();
    const codeChallenge = computeCodeChallenge(codeVerifier);
    const state = generateRandomString(32);

    const authorizeUrl = new URL(`https://${AUTH0_DOMAIN}/authorize`);
    authorizeUrl.searchParams.set('response_type', 'code');
    authorizeUrl.searchParams.set('client_id', AUTH0_CLIENT_ID);
    authorizeUrl.searchParams.set('redirect_uri', AUTH0_REDIRECT_URI);
    authorizeUrl.searchParams.set('scope', AUTH0_SCOPES);
    authorizeUrl.searchParams.set('connection', AUTH0_CONNECTION);
    authorizeUrl.searchParams.set('code_challenge', codeChallenge);
    authorizeUrl.searchParams.set('code_challenge_method', 'S256');
    authorizeUrl.searchParams.set('state', state);

    const promise = new Promise<UserInfo>((resolve, reject) => {
      this.pending = { codeVerifier, state, resolve, reject };
    });

    const opened = await vscode.env.openExternal(
      vscode.Uri.parse(authorizeUrl.toString())
    );
    if (!opened) {
      this.pending = undefined;
      throw new Error('Failed to open browser for Auth0 login.');
    }

    const timeout = setTimeout(
      () => {
        if (this.pending) {
          this.pending.reject(new Error('Login timed out after 5 minutes.'));
          this.pending = undefined;
        }
      },
      5 * 60 * 1000
    );

    try {
      return await promise;
    } finally {
      clearTimeout(timeout);
    }
  }

  public async logout(): Promise<void> {
    const ctx = this.requireContext();
    await Promise.all([
      ctx.secrets.delete(AUTH_SECRET_KEYS.accessToken),
      ctx.secrets.delete(AUTH_SECRET_KEYS.idToken),
      ctx.secrets.delete(AUTH_SECRET_KEYS.refreshToken),
      ctx.secrets.delete(AUTH_SECRET_KEYS.userInfo)
    ]);
    this._onDidChangeSession.fire(undefined);
  }

  public async handleUri(uri: vscode.Uri): Promise<void> {
    if (uri.path !== '/auth-callback') {
      return;
    }
    if (!this.pending) {
      return;
    }
    const pending = this.pending;
    this.pending = undefined;

    const params = new URLSearchParams(uri.query);
    const error = params.get('error');
    if (error) {
      const desc = params.get('error_description') ?? '';
      pending.reject(new Error(`Auth0 error: ${error} ${desc}`.trim()));
      return;
    }

    const code = params.get('code');
    const returnedState = params.get('state');
    if (!code || !returnedState) {
      pending.reject(new Error('Auth0 callback missing code or state.'));
      return;
    }
    if (returnedState !== pending.state) {
      pending.reject(new Error('Auth0 state mismatch — possible CSRF.'));
      return;
    }

    try {
      const tokens = await exchangeCodeForTokens(code, pending.codeVerifier);
      const userInfo = await fetchUserInfo(tokens.access_token);
      await this.persistSession(tokens, userInfo);
      this._onDidChangeSession.fire(userInfo);
      pending.resolve(userInfo);
    } catch (err) {
      pending.reject(err as Error);
    }
  }

  private async persistSession(
    tokens: TokenResponse,
    userInfo: UserInfo
  ): Promise<void> {
    const ctx = this.requireContext();
    await ctx.secrets.store(AUTH_SECRET_KEYS.accessToken, tokens.access_token);
    if (tokens.id_token) {
      await ctx.secrets.store(AUTH_SECRET_KEYS.idToken, tokens.id_token);
    }
    if (tokens.refresh_token) {
      await ctx.secrets.store(
        AUTH_SECRET_KEYS.refreshToken,
        tokens.refresh_token
      );
    }
    await ctx.secrets.store(
      AUTH_SECRET_KEYS.userInfo,
      JSON.stringify(userInfo)
    );
  }

  private requireContext(): vscode.ExtensionContext {
    if (!this.context) {
      throw new Error('AuthService.initialize() was not called.');
    }
    return this.context;
  }
}

interface TokenResponse {
  access_token: string;
  id_token?: string;
  refresh_token?: string;
  token_type: string;
  expires_in: number;
}

async function exchangeCodeForTokens(
  code: string,
  codeVerifier: string
): Promise<TokenResponse> {
  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    client_id: AUTH0_CLIENT_ID,
    code,
    code_verifier: codeVerifier,
    redirect_uri: AUTH0_REDIRECT_URI
  });

  const res = await fetch(`https://${AUTH0_DOMAIN}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: body.toString()
  });
  if (!res.ok) {
    throw new Error(`Token exchange failed: ${res.status} ${await res.text()}`);
  }
  return (await res.json()) as TokenResponse;
}

async function fetchUserInfo(accessToken: string): Promise<UserInfo> {
  const res = await fetch(`https://${AUTH0_DOMAIN}/userinfo`, {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  if (!res.ok) {
    throw new Error(`userinfo failed: ${res.status} ${await res.text()}`);
  }
  return (await res.json()) as UserInfo;
}

function generateCodeVerifier(): string {
  return base64UrlEncode(crypto.randomBytes(32));
}

function computeCodeChallenge(verifier: string): string {
  return base64UrlEncode(crypto.createHash('sha256').update(verifier).digest());
}

function generateRandomString(bytes: number): string {
  return base64UrlEncode(crypto.randomBytes(bytes));
}

function base64UrlEncode(buf: Buffer): string {
  return buf
    .toString('base64')
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '');
}
