export const AUTH0_DOMAIN = 'dev-4xo8tnjk8mvg520h.ca.auth0.com';
export const AUTH0_CLIENT_ID = 'YSaYBjUOsmWQf4J0EcW0BqLb5zDf1NlQ';

export const AUTH0_REDIRECT_URI =
  'vscode://codepathfinder.secureflow/auth-callback';

export const AUTH0_SCOPES = 'openid profile email offline_access';

export type Auth0Connection = 'github' | 'google-oauth2' | 'windowslive';

export const AUTH_SECRET_KEYS = {
  accessToken: 'codePathfinder.auth.accessToken',
  idToken: 'codePathfinder.auth.idToken',
  refreshToken: 'codePathfinder.auth.refreshToken',
  userInfo: 'codePathfinder.auth.userInfo'
} as const;

export const AUTH_STATE_KEYS = {
  guest: 'codePathfinder.auth.guest'
} as const;
