/**
 * DefectDojo Formatter
 * Converts SecureFlow scan results to DefectDojo Generic Findings Import format
 * 
 * DefectDojo JSON format specification:
 * https://docs.defectdojo.com/en/connecting_your_tools/parsers/file/generic/
 */

class DefectDojoFormatter {
  /**
   * Convert SecureFlow scan results to DefectDojo format
   * @param {Object} scanResult - SecureFlow scan result object
   * @returns {Object} DefectDojo formatted findings
   */
  static formatFindings(scanResult) {
    const findings = scanResult.issues.map(issue => {
      return DefectDojoFormatter._convertIssueToFinding(issue, scanResult);
    });

    return {
      findings: findings
    };
  }

  /**
   * Convert a single SecureFlow issue to DefectDojo finding format
   * @param {Object} issue - SecureFlow security issue
   * @param {Object} scanResult - Full scan result for context
   * @returns {Object} DefectDojo finding object
   */
  static _convertIssueToFinding(issue, scanResult) {
    const finding = {
      // Required fields
      title: issue.title || 'Security Issue',
      severity: DefectDojoFormatter._mapSeverity(issue.severity),
      description: issue.description || 'No description provided',
      
      // Optional but recommended fields
      date: scanResult.timestamp ? new Date(scanResult.timestamp).toISOString().split('T')[0] : new Date().toISOString().split('T')[0],
      mitigation: issue.recommendation || 'No mitigation provided',
      
      // Additional context fields
      active: true,
      verified: false,
      false_p: false,
      static_finding: true, // SecureFlow performs static analysis
      
      // Tool identification
      unique_id_from_tool: DefectDojoFormatter._generateUniqueId(issue, scanResult),
      scanner_confidence: DefectDojoFormatter._mapConfidence(issue.severity),
      
      // Tags for categorization
      tags: DefectDojoFormatter._generateTags(issue, scanResult)
    };

    // Add file path and line if available (extracted from description or title)
    const fileInfo = DefectDojoFormatter._extractFileInfo(issue);
    if (fileInfo.filePath) {
      finding.file_path = fileInfo.filePath;
    }
    if (fileInfo.line) {
      finding.line = fileInfo.line;
    }

    // Add CWE if identifiable from the issue
    const cwe = DefectDojoFormatter._extractCWE(issue);
    if (cwe) {
      finding.cwe = cwe;
    }

    // Add CVE if identifiable from the issue
    const cve = DefectDojoFormatter._extractCVE(issue);
    if (cve) {
      finding.cve = cve;
    }

    return finding;
  }

  /**
   * Map SecureFlow severity to DefectDojo severity
   * @param {string} severity - SecureFlow severity
   * @returns {string} DefectDojo severity
   */
  static _mapSeverity(severity) {
    if (!severity) return 'Medium';
    
    const severityMap = {
      'critical': 'Critical',
      'high': 'High',
      'medium': 'Medium',
      'low': 'Low',
      'info': 'Info',
      'informational': 'Info'
    };

    return severityMap[severity.toLowerCase()] || 'Medium';
  }

  /**
   * Map severity to scanner confidence (0-100)
   * @param {string} severity - Issue severity
   * @returns {number} Confidence score
   */
  static _mapConfidence(severity) {
    const confidenceMap = {
      'critical': 95,
      'high': 85,
      'medium': 75,
      'low': 65,
      'info': 50
    };

    return confidenceMap[severity?.toLowerCase()] || 75;
  }

  /**
   * Generate unique ID for the finding
   * @param {Object} issue - Security issue
   * @param {Object} scanResult - Scan result context
   * @returns {string} Unique identifier
   */
  static _generateUniqueId(issue, scanResult) {
    const projectName = scanResult.projectPath ? 
      scanResult.projectPath.split('/').pop() : 'unknown';
    const issueHash = DefectDojoFormatter._simpleHash(
      `${issue.title}-${issue.description}-${issue.severity}`
    );
    return `secureflow-${projectName}-${issueHash}`;
  }

  /**
   * Generate tags for the finding
   * @param {Object} issue - Security issue
   * @param {Object} scanResult - Scan result context
   * @returns {Array<string>} Tags array
   */
  static _generateTags(issue, scanResult) {
    const tags = ['secureflow', 'static-analysis'];
    
    // Add severity as tag
    if (issue.severity) {
      tags.push(`severity-${issue.severity.toLowerCase()}`);
    }

    // Add model used as tag
    if (scanResult.model) {
      tags.push(`model-${scanResult.model.replace(/[^a-zA-Z0-9]/g, '-').toLowerCase()}`);
    }

    // Add language/technology tags based on file extensions or content
    const techTags = DefectDojoFormatter._extractTechnologyTags(issue);
    tags.push(...techTags);

    return tags;
  }

  /**
   * Extract file path and line number from issue content
   * @param {Object} issue - Security issue
   * @returns {Object} File information {filePath, line}
   */
  static _extractFileInfo(issue) {
    const fileInfo = { filePath: null, line: null };
    
    // Common patterns for file references in security issues
    const filePatterns = [
      // Pattern: "in file.js:123" or "file.js line 123"
      /(?:in\s+|file\s+)([^\s:]+\.(?:js|ts|py|go|java|cpp|c|php|rb|cs|swift|kt|scala|rs|dart|vue|jsx|tsx))(?::(\d+)|\s+line\s+(\d+))/i,
      // Pattern: "file.js:123"
      /([^\s:]+\.(?:js|ts|py|go|java|cpp|c|php|rb|cs|swift|kt|scala|rs|dart|vue|jsx|tsx)):(\d+)/i,
      // Pattern: "./path/to/file.js"
      /(\.?\/[^\s]+\.(?:js|ts|py|go|java|cpp|c|php|rb|cs|swift|kt|scala|rs|dart|vue|jsx|tsx))/i
    ];

    const searchText = `${issue.title} ${issue.description}`;
    
    for (const pattern of filePatterns) {
      const match = searchText.match(pattern);
      if (match) {
        fileInfo.filePath = match[1];
        fileInfo.line = parseInt(match[2] || match[3]) || null;
        break;
      }
    }

    return fileInfo;
  }

  /**
   * Extract CWE number from issue content
   * @param {Object} issue - Security issue
   * @returns {number|null} CWE number
   */
  static _extractCWE(issue) {
    const searchText = `${issue.title} ${issue.description}`;
    const cweMatch = searchText.match(/CWE[-\s]?(\d+)/i);
    return cweMatch ? parseInt(cweMatch[1]) : null;
  }

  /**
   * Extract CVE identifier from issue content
   * @param {Object} issue - Security issue
   * @returns {string|null} CVE identifier
   */
  static _extractCVE(issue) {
    const searchText = `${issue.title} ${issue.description}`;
    const cveMatch = searchText.match(/CVE-\d{4}-\d+/i);
    return cveMatch ? cveMatch[0].toUpperCase() : null;
  }

  /**
   * Extract technology tags from issue content
   * @param {Object} issue - Security issue
   * @returns {Array<string>} Technology tags
   */
  static _extractTechnologyTags(issue) {
    const tags = [];
    const searchText = `${issue.title} ${issue.description}`.toLowerCase();
    
    const techMap = {
      'javascript': ['javascript', 'js', 'node.js', 'nodejs', 'npm'],
      'typescript': ['typescript', 'ts'],
      'python': ['python', 'py', 'django', 'flask'],
      'go': ['golang', 'go'],
      'java': ['java', 'spring', 'maven', 'gradle'],
      'csharp': ['c#', 'csharp', '.net', 'dotnet'],
      'php': ['php', 'laravel', 'symfony'],
      'ruby': ['ruby', 'rails'],
      'cpp': ['c++', 'cpp'],
      'rust': ['rust', 'cargo'],
      'swift': ['swift', 'ios'],
      'kotlin': ['kotlin', 'android'],
      'sql': ['sql', 'mysql', 'postgresql', 'sqlite'],
      'docker': ['docker', 'dockerfile', 'container'],
      'kubernetes': ['kubernetes', 'k8s', 'kubectl'],
      'aws': ['aws', 'amazon', 's3', 'ec2', 'lambda'],
      'react': ['react', 'jsx'],
      'vue': ['vue', 'vuejs'],
      'angular': ['angular']
    };

    for (const [tag, keywords] of Object.entries(techMap)) {
      if (keywords.some(keyword => searchText.includes(keyword))) {
        tags.push(tag);
      }
    }

    return tags;
  }

  /**
   * Simple hash function for generating unique IDs
   * @param {string} str - String to hash
   * @returns {string} Hash string
   */
  static _simpleHash(str) {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32-bit integer
    }
    return Math.abs(hash).toString(16);
  }

  /**
   * Validate DefectDojo findings format
   * @param {Object} findings - DefectDojo findings object
   * @returns {Object} Validation result {valid, errors}
   */
  static validateFindings(findings) {
    const errors = [];
    
    if (!findings || typeof findings !== 'object') {
      errors.push('Findings must be an object');
      return { valid: false, errors };
    }

    if (!Array.isArray(findings.findings)) {
      errors.push('findings.findings must be an array');
      return { valid: false, errors };
    }

    findings.findings.forEach((finding, index) => {
      // Check required fields
      if (!finding.title) {
        errors.push(`Finding ${index}: title is required`);
      }
      if (!finding.severity) {
        errors.push(`Finding ${index}: severity is required`);
      } else if (!['Critical', 'High', 'Medium', 'Low', 'Info'].includes(finding.severity)) {
        errors.push(`Finding ${index}: severity must be one of Critical, High, Medium, Low, Info`);
      }
      if (!finding.description) {
        errors.push(`Finding ${index}: description is required`);
      }
    });

    return { valid: errors.length === 0, errors };
  }
}

module.exports = { DefectDojoFormatter };
