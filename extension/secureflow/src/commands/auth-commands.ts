import * as vscode from 'vscode';
import { AuthService, UserInfo } from '../services/auth-service';

export function registerAuthCommands(context: vscode.ExtensionContext): void {
  const auth = AuthService.getInstance();

  context.subscriptions.push(
    vscode.commands.registerCommand('secureflow.login', async () => {
      try {
        const user = await vscode.window.withProgress(
          {
            location: vscode.ProgressLocation.Notification,
            title: 'Signing in with GitHub via Auth0…',
            cancellable: false
          },
          () => auth.login()
        );
        vscode.window.showInformationMessage(
          `Signed in as ${formatUser(user)}.`
        );
      } catch (err) {
        vscode.window.showErrorMessage(
          `Sign-in failed: ${(err as Error).message}`
        );
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('secureflow.logout', async () => {
      const current = await auth.getCurrentUser();
      if (!current) {
        vscode.window.showInformationMessage('You are not signed in.');
        return;
      }
      await auth.logout();
      vscode.window.showInformationMessage('Signed out.');
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('secureflow.showAccount', async () => {
      const current = await auth.getCurrentUser();
      if (!current) {
        const pick = await vscode.window.showInformationMessage(
          'Not signed in.',
          'Sign in'
        );
        if (pick === 'Sign in') {
          await vscode.commands.executeCommand('secureflow.login');
        }
        return;
      }
      const pick = await vscode.window.showInformationMessage(
        `Signed in as ${formatUser(current)}.`,
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
  const render = (user: UserInfo | undefined) => {
    if (user) {
      item.text = `$(github) ${formatUser(user)}`;
      item.tooltip = 'Code Pathfinder — signed in. Click to manage.';
    } else {
      item.text = '$(github) Sign in';
      item.tooltip = 'Code Pathfinder — sign in with GitHub via Auth0';
    }
    item.show();
  };

  context.subscriptions.push(auth.onDidChangeSession(render));
  auth.getCurrentUser().then(render);

  return item;
}

function formatUser(user: UserInfo): string {
  return user.nickname || user.name || user.email || user.sub;
}
