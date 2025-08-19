// Type declarations for SecureFlow CLI Git utilities (CommonJS)

export interface GitChangeInfo {
  filePath: string;
  startLine: number;
  lineCount: number;
  content: string;
}

export function executeCommand(command: string, cwd: string): Promise<string>;
export function parseGitDiff(diffOutput: string, repoPath: string): GitChangeInfo[];
export function getGitChangesAtRepo(
  repoPath: string,
  opts?: { staged?: boolean; unstaged?: boolean }
): Promise<GitChangeInfo[]>;
