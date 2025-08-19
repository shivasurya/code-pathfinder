// SecureFlow CLI Git utilities (CommonJS, VS Code-agnostic)
// Provides helpers to fetch and parse git diffs for staged/unstaged changes.

const cp = require('child_process');
const path = require('path');

/**
 * Execute a shell command and return stdout
 * @param {string} command
 * @param {string} cwd
 * @returns {Promise<string>}
 */
function executeCommand(command, cwd) {
  return new Promise((resolve, reject) => {
    cp.exec(command, { cwd }, (error, stdout, stderr) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(stdout);
    });
  });
}

/**
 * Parse unified diff output (with --unified=0) into change hunks.
 * Only collects added lines (+) per hunk.
 * @param {string} diffOutput
 * @param {string} repoPath
 * @returns {Array<{filePath: string, startLine: number, lineCount: number, content: string}>}
 */
function parseGitDiff(diffOutput, repoPath) {
  const changes = [];
  if (!diffOutput || !diffOutput.trim()) {return changes;}

  const lines = diffOutput.split('\n');
  let i = 0;
  let currentFile = null;

  while (i < lines.length) {
    const line = lines[i];

    if (line.startsWith('diff --git')) {
      const match = line.match(/diff --git a\/(.*?) b\/(.*)/);
      if (match && match[2]) {
        currentFile = match[2].trim();
      }
      i++;
      continue;
    }

    if (line.startsWith('@@') && currentFile) {
      const match = line.match(/@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/);
      if (match) {
        const startLine = parseInt(match[3], 10);
        let addedLines = 0;
        let content = '';
        i++;
        while (i < lines.length && !lines[i].startsWith('diff --git') && !lines[i].startsWith('@@')) {
          const hunkLine = lines[i];
          if (hunkLine.startsWith('+') && !hunkLine.startsWith('+++')) {
            content += hunkLine.substring(1) + '\n';
            addedLines++;
          }
          i++;
        }

        if (addedLines > 0) {
          changes.push({
            filePath: path.join(repoPath, currentFile),
            startLine,
            lineCount: addedLines,
            content: content.trim()
          });
        }
        continue;
      }
    }

    i++;
  }

  return changes;
}

/**
 * Get git changes for a repository by collecting both staged and unstaged diffs.
 * @param {string} repoPath
 * @param {{ staged?: boolean, unstaged?: boolean }} [opts]
 * @returns {Promise<Array<{filePath: string, startLine: number, lineCount: number, content: string}>>}
 */
async function getGitChangesAtRepo(repoPath, opts = {}) {
  const { staged = true, unstaged = true } = opts;
  const all = [];

  if (staged) {
    const out = await executeCommand('git diff --cached --unified=0 --no-color', repoPath);
    all.push(...parseGitDiff(out, repoPath));
  }
  if (unstaged) {
    const out = await executeCommand('git diff --unified=0 --no-color', repoPath);
    all.push(...parseGitDiff(out, repoPath));
  }

  return all;
}

module.exports = {
  executeCommand,
  parseGitDiff,
  getGitChangesAtRepo
};
