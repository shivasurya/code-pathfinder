import * as vscode from 'vscode';
import { SecurityIssue } from '../models/security-issue';
import { AIClientFactory } from '../clients/ai-client-factory';
import { AIModel } from '../settings/settings-manager';

/**
 * Analyzes code for security issues using AI models
 * @param code The code to analyze
 * @param aiModel The AI model to use
 * @param apiKey The API key for the AI model
 * @returns Array of security issues found
 */
export async function analyzeSecurityWithAI(
    code: string, 
    aiModel: AIModel,
    apiKey: string
): Promise<SecurityIssue[]> {
    try {
        // Get the appropriate AI client
        const aiClient = AIClientFactory.getClient(aiModel);
        
        // Construct the prompt for security analysis
        const prompt = `
            You are a security expert analyzing code for vulnerabilities. 
            Review the following code for security issues and return a JSON array of issues found.
            Each issue should have the following format:
            {
                "title": "Issue title",
                "severity": "Low|Medium|High|Critical",
                "description": "Detailed description of the issue",
                "recommendation": "How to fix the issue"
            }
            
            If no issues are found, return an empty array.
            
            Here's the code to analyze:
            
            \`\`\`
            ${code}
            \`\`\`
            
            Please provide your response as valid JSON only, with no additional text.
        `;
        
        // Send the request to the AI model
        const response = await aiClient.sendRequest(prompt, {
            apiKey,
            temperature: 0, // Lower temperature for more consistent results
            maxTokens: 2000  // Allow enough tokens for a detailed analysis
        });
        
        // Parse the response as JSON
        try {
            // Extract the JSON array from the response
            const responseText = response.content.trim();
            const jsonStartIndex = responseText.indexOf('[');
            const jsonEndIndex = responseText.lastIndexOf(']') + 1;
            
            if (jsonStartIndex !== -1 && jsonEndIndex !== -1) {
                const jsonStr = responseText.substring(jsonStartIndex, jsonEndIndex);
                const issues = JSON.parse(jsonStr) as SecurityIssue[];
                return issues;
            }
            
            // Fallback to parsing the entire response if JSON markers not found
            return JSON.parse(responseText) as SecurityIssue[];
            
        } catch (parseError) {
            console.error('Error parsing AI response:', parseError);
            return [{
                title: 'Error Analyzing Code',
                severity: 'Medium',
                description: `The AI response could not be parsed: ${parseError}`,
                recommendation: 'Try again or use a different AI model.'
            }];
        }
    } catch (error) {
        console.error('Error analyzing security with AI:', error);
        return [{
            title: 'AI Analysis Error',
            severity: 'Medium',
            description: `An error occurred while analyzing code with the AI model: ${error}`,
            recommendation: 'Check your API key and internet connection and try again.'
        }];
    }
}

/**
 * Analyzes code for security issues using AI models with streaming response
 * @param code The code to analyze
 * @param aiModel The AI model to use
 * @param apiKey The API key for the AI model
 * @param progressCallback Callback function for progress updates
 * @returns Array of security issues found
 */
export async function analyzeSecurityWithAIStreaming(
    code: string,
    aiModel: AIModel,
    apiKey: string,
    progressCallback: (message: string) => void
): Promise<SecurityIssue[]> {
    return new Promise((resolve, reject) => {
        try {
            // Get the appropriate AI client
            const aiClient = AIClientFactory.getClient(aiModel);
            
            // Construct the prompt for security analysis
            const prompt = `
                You are a security expert analyzing code for vulnerabilities. 
                Review the following code for security issues and return a JSON array of issues found.
                Each issue should have the following format:
                {
                    "title": "Issue title",
                    "severity": "Low|Medium|High|Critical",
                    "description": "Detailed description of the issue",
                    "recommendation": "How to fix the issue"
                }
                
                If no issues are found, return an empty array.
                
                Here's the code to analyze:
                
                \`\`\`
                ${code}
                \`\`\`
                
                Please provide your response as valid JSON only, with no additional text.
            `;
            
            let responseText = '';
            
            // Send the streaming request to the AI model
            aiClient.sendStreamingRequest(
                prompt,
                (chunk) => {
                    responseText = chunk.content;
                    progressCallback(chunk.content);
                    
                    if (chunk.isComplete) {
                        try {
                            // Extract the JSON array from the response
                            const jsonStartIndex = responseText.indexOf('[');
                            const jsonEndIndex = responseText.lastIndexOf(']') + 1;
                            
                            if (jsonStartIndex !== -1 && jsonEndIndex !== -1) {
                                const jsonStr = responseText.substring(jsonStartIndex, jsonEndIndex);
                                const issues = JSON.parse(jsonStr) as SecurityIssue[];
                                resolve(issues);
                                return;
                            }
                            
                            // Fallback to parsing the entire response if JSON markers not found
                            const issues = JSON.parse(responseText) as SecurityIssue[];
                            resolve(issues);
                        } catch (parseError) {
                            console.error('Error parsing AI response:', parseError);
                            resolve([{
                                title: 'Error Analyzing Code',
                                severity: 'Medium',
                                description: `The AI response could not be parsed: ${parseError}`,
                                recommendation: 'Try again or use a different AI model.'
                            }]);
                        }
                    }
                },
                {
                    apiKey,
                    temperature: 0.3,
                    maxTokens: 2000
                }
            ).catch(error => {
                console.error('Error in streaming request:', error);
                reject(error);
            });
            
        } catch (error) {
            console.error('Error analyzing security with AI streaming:', error);
            resolve([{
                title: 'AI Analysis Error',
                severity: 'Medium',
                description: `An error occurred while analyzing code with the AI model: ${error}`,
                recommendation: 'Check your API key and internet connection and try again.'
            }]);
        }
    });
}
