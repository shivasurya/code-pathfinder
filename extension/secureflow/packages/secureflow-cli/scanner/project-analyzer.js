const fs = require('fs');
const path = require('path');
const { promisify } = require('util');

const readdir = promisify(fs.readdir);
const stat = promisify(fs.stat);
const readFile = promisify(fs.readFile);

/**
 * Analyzes project directory structure and provides file listings
 */
class ProjectAnalyzer {
  constructor(projectPath, options = {}) {
    this.projectPath = path.resolve(projectPath);
    this.maxDepth = options.maxDepth || 5;
    this.ignorePatterns = options.ignorePatterns || [
      'node_modules',
      '.git',
      '.vscode',
      '.idea',
      'dist',
      'build',
      'coverage',
      '.nyc_output',
      'logs',
      '*.log',
      '.DS_Store',
      'Thumbs.db'
    ];
  }

  /**
   * Get project directory structure as a tree
   */
  async getDirectoryStructure() {
    const structure = await this._buildDirectoryTree(this.projectPath, 0);
    return structure;
  }

  /**
   * Get all files in the project with metadata
   */
  async getAllFiles() {
    const files = [];
    await this._collectFiles(this.projectPath, files, 0);
    return files;
  }

  /**
   * Check if a file should be ignored
   */
  _shouldIgnore(filePath, fileName) {
    // Check if it's a hidden file (starts with .)
    if (fileName.startsWith('.') && fileName !== '.env' && fileName !== '.gitignore') {
      return true;
    }

    // Check against ignore patterns
    for (const pattern of this.ignorePatterns) {
      if (pattern.includes('*')) {
        const regex = new RegExp(pattern.replace(/\*/g, '.*'));
        if (regex.test(fileName) || regex.test(filePath)) {
          return true;
        }
      } else if (fileName === pattern || filePath.includes(pattern)) {
        return true;
      }
    }

    return false;
  }

  /**
   * Check if a path is a symlink
   */
  async _isSymlink(filePath) {
    try {
      const stats = await stat(filePath);
      return stats.isSymbolicLink();
    } catch (error) {
      return false;
    }
  }

  /**
   * Check if file is within project scope
   */
  _isWithinProjectScope(filePath) {
    const resolvedPath = path.resolve(filePath);
    const projectPath = path.resolve(this.projectPath);
    return resolvedPath.startsWith(projectPath);
  }

  /**
   * Get file line count
   */
  async _getLineCount(filePath) {
    try {
      const content = await readFile(filePath, 'utf8');
      return content.split('\n').length;
    } catch (error) {
      return 0;
    }
  }

  /**
   * Build directory tree recursively
   */
  async _buildDirectoryTree(dirPath, depth) {
    if (depth > this.maxDepth) {
      return { name: path.basename(dirPath), type: 'directory', truncated: true };
    }

    const items = [];
    try {
      const entries = await readdir(dirPath);
      
      for (const entry of entries) {
        const fullPath = path.join(dirPath, entry);
        
        if (this._shouldIgnore(fullPath, entry)) {
          continue;
        }

        if (await this._isSymlink(fullPath)) {
          continue;
        }

        if (!this._isWithinProjectScope(fullPath)) {
          continue;
        }

        try {
          const stats = await stat(fullPath);
          
          if (stats.isDirectory()) {
            const subTree = await this._buildDirectoryTree(fullPath, depth + 1);
            items.push(subTree);
          } else if (stats.isFile()) {
            const lineCount = await this._getLineCount(fullPath);
            items.push({
              name: entry,
              type: 'file',
              path: fullPath,
              size: stats.size,
              lineCount,
              extension: path.extname(entry)
            });
          }
        } catch (error) {
          // Skip files we can't access
          continue;
        }
      }
    } catch (error) {
      return { name: path.basename(dirPath), type: 'directory', error: error.message };
    }

    return {
      name: path.basename(dirPath),
      type: 'directory',
      path: dirPath,
      children: items
    };
  }

  /**
   * Collect all files recursively
   */
  async _collectFiles(dirPath, files, depth) {
    if (depth > this.maxDepth) {
      return;
    }

    try {
      const entries = await readdir(dirPath);
      
      for (const entry of entries) {
        const fullPath = path.join(dirPath, entry);
        
        if (this._shouldIgnore(fullPath, entry)) {
          continue;
        }

        if (await this._isSymlink(fullPath)) {
          continue;
        }

        if (!this._isWithinProjectScope(fullPath)) {
          continue;
        }

        try {
          const stats = await stat(fullPath);
          
          if (stats.isDirectory()) {
            await this._collectFiles(fullPath, files, depth + 1);
          } else if (stats.isFile()) {
            const lineCount = await this._getLineCount(fullPath);
            const relativePath = path.relative(this.projectPath, fullPath);
            
            files.push({
              name: entry,
              path: fullPath,
              relativePath,
              size: stats.size,
              lineCount,
              extension: path.extname(entry),
              directory: path.dirname(relativePath)
            });
          }
        } catch (error) {
          // Skip files we can't access
          continue;
        }
      }
    } catch (error) {
      // Skip directories we can't access
      return;
    }
  }

  /**
   * Generate a summary of the project structure for LLM
   */
  async getProjectSummary() {
    const structure = await this.getDirectoryStructure();
    const files = await this.getAllFiles();
    
    const summary = {
      projectPath: this.projectPath,
      totalFiles: files.length,
      filesByExtension: this._groupFilesByExtension(files),
      directoryStructure: this._formatStructureForLLM(structure),
      importantFiles: this._identifyImportantFiles(files)
    };

    return summary;
  }

  /**
   * Group files by extension for analysis
   */
  _groupFilesByExtension(files) {
    const groups = {};
    files.forEach(file => {
      const ext = file.extension || 'no-extension';
      if (!groups[ext]) {
        groups[ext] = [];
      }
      groups[ext].push(file.relativePath);
    });
    return groups;
  }

  /**
   * Format directory structure for LLM consumption
   * Formats paths without project root directory prefix
   * Example:
   * LICENSE Lines: 22
   * README.md Lines: 95
   * cmd/todoist/main.go Lines: 30
   */
  _formatStructureForLLM(structure, parentPath = '') {
    let result = '';
    
    if (structure.type === 'file') {
      const filePath = parentPath ? `${parentPath}/${structure.name}` : structure.name;
      return `${filePath} Lines: ${structure.lineCount}\n`;
    }
    
    if (structure.children) {
      const currentPath = structure.name === path.basename(this.projectPath) ? '' : 
        parentPath ? `${parentPath}/${structure.name}` : structure.name;
      structure.children.forEach(child => {
        result += this._formatStructureForLLM(child, currentPath);
      });
    }
    
    return result;
  }

  /**
   * Format directory structure for CLI display with tree characters
   */
  formatDirectoryTreeForCLI(structure, prefix = '', isLast = true) {
    let result = '';
    
    if (prefix === '') {
      // Root directory
      result += `📁 ${structure.name}/\n`;
      if (structure.children && structure.children.length > 0) {
        const sortedChildren = this._sortChildren(structure.children);
        sortedChildren.forEach((child, index) => {
          const isLastChild = index === sortedChildren.length - 1;
          result += this._formatTreeNode(child, '', isLastChild);
        });
      }
    } else {
      result += this._formatTreeNode(structure, prefix, isLast);
    }
    
    return result;
  }

  /**
   * Format a single tree node
   */
  _formatTreeNode(node, prefix, isLast) {
    let result = '';
    const connector = isLast ? '└── ' : '├── ';
    const icon = node.type === 'directory' ? '📁' : '📄';
    const name = node.type === 'directory' ? `${node.name}/` : node.name;
    const info = node.type === 'file' && node.lineCount ? ` (${node.lineCount} lines)` : '';
    
    result += `${prefix}${connector}${icon} ${name}${info}\n`;
    
    if (node.children && node.children.length > 0) {
      const sortedChildren = this._sortChildren(node.children);
      const newPrefix = prefix + (isLast ? '    ' : '│   ');
      
      sortedChildren.forEach((child, index) => {
        const isLastChild = index === sortedChildren.length - 1;
        result += this._formatTreeNode(child, newPrefix, isLastChild);
      });
    }
    
    if (node.truncated) {
      const truncatedPrefix = prefix + (isLast ? '    ' : '│   ');
      result += `${truncatedPrefix}└── ... (truncated at max depth)\n`;
    }
    
    return result;
  }

  /**
   * Sort children: directories first, then files, alphabetically
   */
  _sortChildren(children) {
    return children.sort((a, b) => {
      if (a.type === 'directory' && b.type === 'file') {
        return -1;
      }
      if (a.type === 'file' && b.type === 'directory') {
        return 1;
      }
      return a.name.localeCompare(b.name);
    });
  }

  /**
   * Print directory tree to console with proper formatting
   */
  printDirectoryTree(structure) {
    console.log(this.formatDirectoryTreeForCLI(structure));
  }

  /**
   * Identify potentially important files based on common patterns
   */
  _identifyImportantFiles(files) {
    const important = [];
    const importantPatterns = [
      /package\.json$/,
      /requirements\.txt$/,
      /Dockerfile$/,
      /docker-compose\.ya?ml$/,
      /\.env$/,
      /config\.(js|json|yaml|yml)$/,
      /settings\.(py|js|json)$/,
      /main\.(js|py|go|java)$/,
      /index\.(js|html|php)$/,
      /app\.(js|py|go)$/,
      /server\.(js|py|go)$/,
      /README\.md$/,
      /security\.(js|py|go|java)$/,
      /auth\.(js|py|go|java)$/,
      /login\.(js|py|go|java)$/,
      /admin\.(js|py|go|java)$/
    ];

    files.forEach(file => {
      for (const pattern of importantPatterns) {
        if (pattern.test(file.relativePath)) {
          important.push(file);
          break;
        }
      }
    });

    return important;
  }
}

module.exports = { ProjectAnalyzer };
