const fs = require('fs');
const path = require('path');
const { promisify } = require('util');
const { cyan, yellow, red, green, dim } = require('colorette');

const readFile = promisify(fs.readFile);
const stat = promisify(fs.stat);

/**
 * Handles LLM file requests with tool-like interface
 */
class FileRequestHandler {
  constructor(projectPath, options = {}) {
    this.projectPath = path.resolve(projectPath);
    this.maxFileLines = options.maxFileLines || 1000;
    this.partialReadLines = options.partialReadLines || 500;
    this.requestLog = [];
  }

  /**
   * Process LLM file requests from structured response
   */
  async processFileRequests(llmResponse) {
    const requests = this._parseFileRequests(llmResponse);
    const results = [];

    for (const request of requests) {
      console.log(cyan(`🤖 LLM requesting file: ${request.path}`));
      
      const result = await this._handleFileRequest(request);
      results.push(result);
      
      // Log the request
      this.requestLog.push({
        timestamp: new Date().toISOString(),
        request,
        result: {
          status: result.status,
          reason: result.reason,
          linesReturned: result.content ? result.content.split('\n').length : 0
        }
      });
    }

    return results;
  }

  /**
   * Parse file requests from LLM response
   * Expected format: <file_request path="./src/app.js" reason="Check authentication logic" />
   */
  _parseFileRequests(response) {
    const requests = [];
    const fileRequestRegex = /<file_request\s+path="([^"]+)"(?:\s+reason="([^"]*)")?\s*\/>/g;
    
    let match;
    while ((match = fileRequestRegex.exec(response)) !== null) {
      requests.push({
        path: match[1],
        reason: match[2] || 'No reason provided'
      });
    }

    return requests;
  }

  /**
   * Handle individual file request with filtering and validation
   */
  async _handleFileRequest(request) {
    const { path: requestedPath, reason } = request;
    
    try {
      // Resolve and validate path
      const fullPath = this._resolvePath(requestedPath);
      
      // Check if file is within project scope
      if (!this._isWithinProjectScope(fullPath)) {
        console.log(red(`❌ File outside project scope: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'File is outside project directory scope',
          path: requestedPath
        };
      }

      // Check if it's a hidden file
      if (this._isHiddenFile(fullPath)) {
        console.log(yellow(`⚠️  Hidden file ignored: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'Hidden files are ignored',
          path: requestedPath
        };
      }

      // Check if it's a symlink
      if (await this._isSymlink(fullPath)) {
        console.log(yellow(`⚠️  Symlink ignored: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'Symlinks are ignored',
          path: requestedPath
        };
      }

      // Check if file exists
      if (!fs.existsSync(fullPath)) {
        console.log(red(`❌ File not found: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'File does not exist',
          path: requestedPath
        };
      }

      // Read file content with size limits
      const content = await this._readFileWithLimits(fullPath);
      
      console.log(green(`✅ File read: ${requestedPath} (${content.split('\n').length} lines)`));
      console.log(dim(`   Reason: ${reason}`));

      return {
        status: 'success',
        path: requestedPath,
        fullPath,
        content,
        reason,
        lines: content.split('\n').length
      };

    } catch (error) {
      console.log(red(`❌ Error reading file ${requestedPath}: ${error.message}`));
      return {
        status: 'error',
        reason: error.message,
        path: requestedPath
      };
    }
  }

  /**
   * Resolve relative path to absolute path
   */
  _resolvePath(requestedPath) {
    if (path.isAbsolute(requestedPath)) {
      return requestedPath;
    }
    return path.resolve(this.projectPath, requestedPath);
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
   * Check if file is hidden
   */
  _isHiddenFile(filePath) {
    const fileName = path.basename(filePath);
    return fileName.startsWith('.') && fileName !== '.env' && fileName !== '.gitignore';
  }

  /**
   * Check if path is a symlink
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
   * Read file with size limits and partial reading
   */
  async _readFileWithLimits(filePath) {
    const content = await readFile(filePath, 'utf8');
    const lines = content.split('\n');

    if (lines.length <= this.maxFileLines) {
      return content;
    }

    // File is too large, read first portion
    console.log(yellow(`📄 Large file detected (${lines.length} lines), reading first ${this.partialReadLines} lines`));
    
    const partialContent = lines.slice(0, this.partialReadLines).join('\n');
    const truncationNote = `\n\n/* [SECUREFLOW] File truncated - showing first ${this.partialReadLines} of ${lines.length} lines */`;
    
    return partialContent + truncationNote;
  }

  /**
   * Generate file request tools description for LLM
   */
  getFileRequestInstructions() {
    return `
You can request specific files for analysis using the following format:
<file_request path="./relative/path/to/file.js" reason="Brief reason for requesting this file" />

Rules for file requests:
1. Use relative paths from the project root
2. Provide a brief reason for each request
3. Files will be automatically filtered (no hidden files, symlinks, or files outside project scope)
4. Large files (>1000 lines) will be truncated to first 500 lines
5. You'll be notified if a file cannot be read

Example:
<file_request path="./src/auth/login.js" reason="Check authentication implementation" />
<file_request path="./config/database.js" reason="Review database configuration" />
`;
  }

  /**
   * Get request log for debugging
   */
  getRequestLog() {
    return this.requestLog;
  }

  /**
   * Clear request log
   */
  clearRequestLog() {
    this.requestLog = [];
  }
}

module.exports = { FileRequestHandler };
