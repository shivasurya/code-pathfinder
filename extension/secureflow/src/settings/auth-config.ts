export const AUTH0_DOMAIN = 'dev-4xo8tnjk8mvg520h.ca.auth0.com';
export const AUTH0_CLIENT_ID = 'YSaYBjUOsmWQf4J0EcW0BqLb5zDf1NlQ';

export const AUTH0_REDIRECT_URI =
  'vscode://codepathfinder.secureflow/auth-callback';

export const AUTH0_SCOPES = 'openid profile email offline_access';
export const AUTH0_CONNECTION = 'github';

export const AUTH_SECRET_KEYS = {
  accessToken: 'codePathfinder.auth.accessToken',
  idToken: 'codePathfinder.auth.idToken',
  refreshToken: 'codePathfinder.auth.refreshToken',
  userInfo: 'codePathfinder.auth.userInfo'
} as const;
