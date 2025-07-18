import { SecurityIssue } from './models/security-issue';

/**
 * Performs security analysis on the given code snippet
 * @param code The code to analyze
 * @returns Array of security issues found
 */
export function performSecurityAnalysis(code: string): SecurityIssue[] {
    // This is a mock implementation. In a real extension, you would implement actual code analysis.
    const issues: SecurityIssue[] = [];
    
    // Check for SQL injection vulnerability pattern
    if (code.toLowerCase().includes('sql') && code.includes("'") && code.includes("+")) {
        issues.push({
            title: "Potential SQL Injection",
            severity: "High",
            description: "String concatenation used with SQL queries can lead to SQL injection attacks.",
            recommendation: "Use parameterized queries or prepared statements instead of string concatenation."
        });
    }
    
    // Check for potential XSS vulnerability pattern
    if ((code.includes("innerHTML") || code.includes("document.write")) && code.includes("${")) {
        issues.push({
            title: "Potential Cross-Site Scripting (XSS)",
            severity: "High",
            description: "Directly inserting user input into DOM can lead to XSS attacks.",
            recommendation: "Use textContent instead of innerHTML, or sanitize user input before inserting into DOM."
        });
    }
    
    // Check for hardcoded secrets
    if (code.toLowerCase().includes("password") || code.toLowerCase().includes("token") || code.toLowerCase().includes("secret")) {
        if (code.includes("=") && (code.includes("'") || code.includes("\""))) {
            issues.push({
                title: "Hardcoded Secret",
                severity: "Medium",
                description: "Sensitive information appears to be hardcoded in the source code.",
                recommendation: "Use environment variables or a secure vault for storing sensitive information."
            });
        }
    }
    
    // Check for insecure random number generation
    if (code.includes("Math.random(") && (
        code.toLowerCase().includes("auth") || 
        code.toLowerCase().includes("token") || 
        code.toLowerCase().includes("password") || 
        code.toLowerCase().includes("secure")
    )) {
        issues.push({
            title: "Insecure Random Number Generation",
            severity: "Medium",
            description: "Using Math.random() for security-sensitive operations is not recommended.",
            recommendation: "Use a cryptographically secure random number generator like crypto.getRandomValues()."
        });
    }
    
    // Check for potential command injection
    if (code.includes("exec(") || code.includes("spawn(") || code.includes("system(")) {
        if (code.includes("+") || code.includes("`") || code.includes("${")) {
            issues.push({
                title: "Potential Command Injection",
                severity: "Critical",
                description: "Dynamic command execution can lead to command injection vulnerabilities.",
                recommendation: "Avoid using user input in command execution. If necessary, properly validate and sanitize the input."
            });
        }
    }
    
    // If no issues found but code contains security-sensitive patterns, add a mock issue (for demonstration)
    if (issues.length === 0 && (
        code.toLowerCase().includes("security") ||
        code.toLowerCase().includes("login") ||
        code.toLowerCase().includes("auth") ||
        code.toLowerCase().includes("password")
    )) {
        issues.push({
            title: "Potential Weak Authentication",
            severity: "Medium",
            description: "Authentication logic might not follow security best practices.",
            recommendation: "Implement multi-factor authentication and ensure proper password handling."
        });
    }
    
    return issues;
}
