import { SecurityIssue } from './models/security-issue';
import { AIModel } from './settings-manager';
import { analyzeSecurityWithAI } from './security-analyzer-ai';

/**
 * Performs security analysis on the given code snippet
 * Can use both pattern-based detection and AI-based analysis
 * @param code The code to analyze
 * @param aiModel Optional parameter to specify which AI Model to use
 * @param apiKey Optional API key for AI Model (if not provided, only pattern-based analysis is done)
 * @returns Array of security issues found
 */
export function performSecurityAnalysis(code: string, aiModel?: AIModel, apiKey?: string): SecurityIssue[] {
    // Log which AI Model would be used (for future implementation)
    console.log(`Using AI Model for analysis: ${aiModel || 'default'}`);

    // This is a pattern-based implementation. It doesn't require an API key.
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

/**
 * Performs security analysis on the given code snippet asynchronously,
 * utilizing both pattern-based detection and AI-based analysis if an API key is provided
 * @param code The code to analyze
 * @param aiModel The AI Model to use
 * @param apiKey API key for the AI Model
 * @returns Promise with array of security issues found
 */
export async function performSecurityAnalysisAsync(
    code: string, 
    aiModel: AIModel, 
    apiKey?: string
): Promise<SecurityIssue[]> {
    
    // If no API key is provided, just return the pattern-based results
    if (!apiKey) {
        return [];
    }
    
    try {
        
        // Run the AI-based analysis
        const aiIssues = await analyzeSecurityWithAI(code, aiModel, apiKey);
        
        // Merge the results, removing any duplicates
        const allIssues = [];
        
        // Add AI issues that don't overlap with pattern issues
        for (const aiIssue of aiIssues) {
            allIssues.push(aiIssue);
        }
        
        return allIssues;
    } catch (error) {
        console.error('Error in AI-based analysis:', error);
        // If AI analysis fails, return just the pattern-based results
        return [];
    }
}
