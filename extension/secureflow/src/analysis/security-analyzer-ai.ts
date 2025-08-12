import * as vscode from 'vscode';
import * as path from 'path';
import { SecurityIssue } from '../models/security-issue';
import { AIClientFactory } from '../clients/ai-client-factory';
import { AIModel } from '../settings/settings-manager';
import { ProfileStorageService } from '../services/profile-storage-service';
import { StoredProfile } from '../models/profile-store';
import { loadPrompt } from '../prompts/prompt-loader';

/**
 * Helper function to find the most relevant profile for a file path
 * @param filePath The file path to find a profile for
 * @param allProfiles Array of all available profiles
 * @returns The most relevant profile or undefined
 */
function findRelevantProfile(filePath: string, allProfiles: StoredProfile[]): StoredProfile | undefined {
    if (allProfiles.length === 0) {
        return undefined;
    }

    // Check if file is in a workspace folder
    const workspaceFolder = vscode.workspace.getWorkspaceFolder(vscode.Uri.file(filePath));
    if (!workspaceFolder) {
        return undefined;
    }

    // Find profiles within the workspace
    const relevantProfiles = allProfiles.filter(profile => {
        const profileUri = vscode.Uri.parse(profile.workspaceFolderUri);
        const profileWorkspace = vscode.workspace.getWorkspaceFolder(profileUri);
        return profileWorkspace && profileWorkspace.uri.fsPath === workspaceFolder.uri.fsPath;
    });

    if (relevantProfiles.length === 0) {
        return undefined;
    }

    // Get relative path within workspace
    const relativePath = path.relative(workspaceFolder.uri.fsPath, filePath);

    // Find profile where file path falls under profile path
    return relevantProfiles.find(profile => {
        const normalizedProfilePath = profile.path === '/' ? '' : profile.path;
        const profilePathParts = normalizedProfilePath.split('/').filter(p => p);
        const filePathParts = relativePath.split(path.sep).filter(p => p);

        // Check if file path starts with profile path parts
        return profilePathParts.every((part, index) => filePathParts[index] === part);
    });
}

/**
 * Analyzes code for security issues using AI models
 * @param code The code to analyze
 * @param aiModel The AI model to use
 * @param apiKey The API key for the AI model
 * @param filePath Optional file path to get relevant profile context
 * @param context Optional VS Code extension context for profile service
 * @returns Array of security issues found
 */
export async function analyzeSecurityWithAI(
    code: string, 
    aiModel: AIModel,
    apiKey: string,
    filePath?: string,
    context?: vscode.ExtensionContext
): Promise<SecurityIssue[]> {
    try {
        // Get the appropriate AI client
        const aiClient = AIClientFactory.getClient(aiModel);

        // Load the base prompt template
        let consolidatedReviewContent = await loadPrompt('common/review-changes.txt');
        
        // Get the profile context and add it to the prompt
        if (filePath && context) {
            const profileService = new ProfileStorageService(context);
            const profile = findRelevantProfile(filePath, profileService.getAllProfiles());
            
            if (profile) {
                const profileContext = `/*
[SECUREFLOW PROFILE CONTEXT]
Name: ${profile.name}
Category: ${profile.category}
Path: ${profile.path}
Languages: ${profile.languages.join(', ')}
Frameworks: ${profile.frameworks.join(', ')}
Build Tools: ${profile.buildTools.join(', ')}
Evidence: ${profile.evidence.join('; ')}
*/

`;
                
                consolidatedReviewContent += `\n${profileContext}`;
            }
        }
        
        // Add the selected code as if it's a code change/diff
        const fileName = filePath ? path.basename(filePath) : 'selected-code';
        consolidatedReviewContent += `\n=== FILE: ${fileName} ===\n`;
        consolidatedReviewContent += `<SELECTED_CODE>\n${code}\n</SELECTED_CODE>\n`;
        consolidatedReviewContent += `\n=== END FILE: ${fileName} ===\n\n`;
        
        // Use the consolidated content as the prompt
        const prompt = consolidatedReviewContent;

        
        // Send the request to the AI model
        const response = await aiClient.sendRequest(prompt, {
            apiKey,
            temperature: 0, // Lower temperature for more consistent results
            maxTokens: 2000  // Allow enough tokens for a detailed analysis
        });

        console.log(response);
        
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
