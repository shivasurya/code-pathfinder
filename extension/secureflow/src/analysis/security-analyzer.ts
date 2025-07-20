import { SecurityIssue } from '../models/security-issue';
import { AIModel } from '../settings/settings-manager';
import { analyzeSecurityWithAI } from './security-analyzer-ai';

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
