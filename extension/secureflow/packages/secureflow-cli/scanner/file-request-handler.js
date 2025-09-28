const fs = require('fs');
const path = require('path');
const { promisify } = require('util');
const { cyan, yellow, red, green, dim } = require('colorette');
const { loadPrompt } = require('../lib/prompts/prompt-loader');

const readFile = promisify(fs.readFile);
const stat = promisify(fs.stat);

/**
 * Handles LLM file requests with tool-like interface
 */
class FileRequestHandler {
  constructor(projectPath, options = {}) {
    this.projectPath = path.resolve(projectPath);
    this.maxFileLines = 5000;
    this.partialReadLines = 5000;
    this.requestLog = [];
  }

  /**
   * Process LLM file requests from structured response
   */
  async processFileRequests(llmResponse) {
    const requests = this._parseFileRequests(llmResponse);
    const results = [];

    for (const request of requests) {
      console.log(cyan(`Read(${request.path})`));
      console.log(dim(`  └ • Read ${path.basename(request.path)} (${request.reason})`));
      
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
   * Process LLM list file requests from structured response
   */
  async processListFileRequests(llmResponse) {
    const requests = this._parseListFileRequests(llmResponse);
    const results = [];

    for (const request of requests) {
      console.log(cyan(`ListFiles(${request.path})`));
      console.log(dim(`  └ • List directory ${path.basename(request.path)} (${request.reason})`));
      
      const result = await this._handleListFileRequest(request);
      results.push(result);
      
      // Log the request
      this.requestLog.push({
        timestamp: new Date().toISOString(),
        request,
        result: {
          status: result.status,
          reason: result.reason,
          filesListed: result.files ? result.files.length : 0
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
   * Parse list file requests from LLM response
   * Expected format: <list_file_request path="./src" reason="Check directory structure" />
   */
  _parseListFileRequests(response) {
    const requests = [];
    const listFileRequestRegex = /<list_file_request\s+path="([^"]+)"(?:\s+reason="([^"]*)")?\s*\/>/g;
    
    let match;
    while ((match = listFileRequestRegex.exec(response)) !== null) {
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
      
      // console.log(green(`✅ File read: ${requestedPath} (${content.split('\n').length} lines)`));
      // console.log(dim(`   Reason: ${reason}`));

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
   * Handle individual list file request with filtering and validation
   */
  async _handleListFileRequest(request) {
    const { path: requestedPath, reason } = request;
    
    try {
      // Resolve and validate path
      const fullPath = this._resolvePath(requestedPath);
      
      // Check if directory is within project scope
      if (!this._isWithinProjectScope(fullPath)) {
        console.log(red(`❌ Directory outside project scope: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'Directory is outside project directory scope',
          path: requestedPath
        };
      }

      // Check if it's a hidden directory
      if (this._isHiddenFile(fullPath)) {
        console.log(yellow(`⚠️  Hidden directory ignored: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'Hidden directories are ignored',
          path: requestedPath
        };
      }

      // Check if directory exists
      if (!fs.existsSync(fullPath)) {
        console.log(red(`❌ Directory not found: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'Directory does not exist',
          path: requestedPath
        };
      }

      // Check if it's actually a directory
      const stats = await stat(fullPath);
      if (!stats.isDirectory()) {
        console.log(red(`❌ Path is not a directory: ${requestedPath}`));
        return {
          status: 'rejected',
          reason: 'Path is not a directory',
          path: requestedPath
        };
      }

      // List directory contents
      const contents = await this._listDirectoryContents(fullPath);
      
      return {
        status: 'success',
        path: requestedPath,
        fullPath,
        files: contents.files,
        directories: contents.directories,
        all: contents.all,
        reason,
        fileCount: contents.files.length,
        directoryCount: contents.directories.length,
        totalCount: contents.all.length
      };

    } catch (error) {
      console.log(red(`❌ Error listing directory ${requestedPath}: ${error.message}`));
      return {
        status: 'error',
        reason: error.message,
        path: requestedPath
      };
    }
  }

  /**
   * List directory contents with filtering, including both files and directories with full paths
   */
  async _listDirectoryContents(dirPath) {
    const { readdir } = require('fs').promises;
    
    try {
      const items = await readdir(dirPath, { withFileTypes: true });
      const result = {
        files: [],
        directories: [],
        all: []
      };
      
      for (const item of items) {
        // Skip hidden files and directories (except important config files)
        if (item.name.startsWith('.') && 
            item.name !== '.env' && 
            item.name !== '.gitignore' && 
            item.name !== '.config' &&
            item.name !== '.npmrc' &&
            item.name !== '.dockerignore') {
          continue;
        }
        
        const itemPath = path.join(dirPath, item.name);
        const relativePath = path.relative(this.projectPath, itemPath);
        const fullPath = path.resolve(itemPath);
        
        const itemInfo = {
          name: item.name,
          type: item.isDirectory() ? 'directory' : 'file',
          relativePath: relativePath.startsWith('.') ? relativePath : `./${relativePath}`,
          fullPath: fullPath
        };
        
        if (item.isDirectory()) {
          result.directories.push(itemInfo);
        } else if (item.isFile()) {
          result.files.push(itemInfo);
        }
        
        result.all.push(itemInfo);
      }
      
      // Sort by type (directories first) then by name
      const sortFn = (a, b) => {
        if (a.type !== b.type) {
          return a.type === 'directory' ? -1 : 1;
        }
        return a.name.localeCompare(b.name);
      };
      
      result.files.sort(sortFn);
      result.directories.sort(sortFn);
      result.all.sort(sortFn);
      
      return result;
    } catch (error) {
      throw new Error(`Failed to read directory: ${error.message}`);
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
  async getFileRequestInstructions() {
    return await loadPrompt('tools/file-request-instructions.txt');
  }

  /**
   * Get List file request instructions
   */
  async getListFileRequestInstructions() {
    return await loadPrompt('tools/list-file-request-instructions.txt');
  }

  /**
   * Get Wordpress specific file request instructions
   */
  async getWordpressFileRequestInstructions() {
    return await loadPrompt('technologies/wordpress-plugins/wordpress.txt');
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
