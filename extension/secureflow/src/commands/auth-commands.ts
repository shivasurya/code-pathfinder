import * as vscode from 'vscode';
import { AuthService, Session, UserInfo } from '../services/auth-service';
import { Auth0Connection } from '../settings/auth-config';

export function registerAuthCommands(context: vscode.ExtensionContext): void {
  const auth = AuthService.getInstance();

  context.subscriptions.push(
    vscode.commands.registerCommand(
      'secureflow.login',
      async (connection?: Auth0Connection) => {
        const provider = connection ?? 'github';
        try {
          const user = await vscode.window.withProgress(
            {
              location: vscode.ProgressLocation.Notification,
              title: `Signing in with ${providerLabel(provider)} via Auth0…`,
              cancellable: false
            },
            () => auth.login(provider)
          );
          vscode.window.showInformationMessage(
            `Signed in as ${formatUser(user)}.`
          );
        } catch (err) {
          vscode.window.showErrorMessage(
            `Sign-in failed: ${(err as Error).message}`
          );
        }
      }
    )
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('secureflow.continueAsGuest', async () => {
      await auth.continueAsGuest();
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('secureflow.logout', async () => {
      const session = await auth.getSession();
      if (session.kind === 'none') {
        vscode.window.showInformationMessage('You are not signed in.');
        return;
      }
      await auth.logout();
      vscode.window.showInformationMessage('Signed out.');
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('secureflow.showAccount', async () => {
      const session = await auth.getSession();
      if (session.kind === 'none') {
        const pick = await vscode.window.showInformationMessage(
          'Not signed in.',
          'Sign in with GitHub'
        );
        if (pick === 'Sign in with GitHub') {
          await vscode.commands.executeCommand('secureflow.login', 'github');
        }
        return;
      }
      const label =
        session.kind === 'user' ? formatUser(session.user) : 'Guest';
      const pick = await vscode.window.showInformationMessage(
        `Signed in as ${label}.`,
        'Sign out'
      );
      if (pick === 'Sign out') {
        await vscode.commands.executeCommand('secureflow.logout');
      }
    })
  );
}

export function createAuthStatusBarItem(
  context: vscode.ExtensionContext
): vscode.StatusBarItem {
  const item = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100
  );
  item.command = 'secureflow.showAccount';
  context.subscriptions.push(item);

  const auth = AuthService.getInstance();
  const render = (session: Session) => {
    switch (session.kind) {
      case 'user':
        item.text = `$(codepathfinder-logo) ${formatUser(session.user)}`;
        item.tooltip = 'Code Pathfinder — signed in. Click to manage.';
        break;
      case 'guest':
        item.text = '$(codepathfinder-logo) Guest';
        item.tooltip = 'Code Pathfinder — browsing as guest. Click to sign in.';
        break;
      case 'none':
        item.text = '$(codepathfinder-logo) Sign in';
        item.tooltip = 'Code Pathfinder — sign in';
        break;
    }
    item.show();
  };

  context.subscriptions.push(auth.onDidChangeSession(render));
  auth.getSession().then(render);

  return item;
}

function providerLabel(connection: Auth0Connection): string {
  switch (connection) {
    case 'github':
      return 'GitHub';
    case 'google-oauth2':
      return 'Google';
    case 'windowslive':
      return 'Microsoft';
  }
}

function formatUser(user: UserInfo): string {
  return user.nickname || user.name || user.email || user.sub;
}
