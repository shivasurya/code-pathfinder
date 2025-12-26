import * as vscode from 'vscode';
import * as path from 'path';
import { SecurityIssue } from '../models/security-issue';
import { AIClientFactory } from '@codepathfinder/secureflow-cli';
import { AIModel } from '../settings/settings-manager';
import { ProfileStorageService } from '../services/profile-storage-service';
import { StoredProfile } from '../models/profile-store';
import { loadPrompt } from '@codepathfinder/secureflow-cli';

/**
 * Parses XML response from AI and extracts security issues
 * @param responseText The raw response text from AI
 * @returns Array of SecurityIssue objects
 */
function parseXMLResponse(responseText: string): SecurityIssue[] {
  // Find the <issues> tag anywhere in the response
  const issuesStartMatch = responseText.match(/<issues\s*>/i);
  const issuesEndMatch = responseText.match(/<\/issues\s*>/i);

  if (!issuesStartMatch || !issuesEndMatch) {
    // If no issues tags found, return empty array (no issues found)
    return [];
  }

  const issuesStartIndex = issuesStartMatch.index! + issuesStartMatch[0].length;
  const issuesEndIndex = issuesEndMatch.index!;

  if (issuesStartIndex >= issuesEndIndex) {
    return [];
  }

  const issuesXml = responseText.substring(issuesStartIndex, issuesEndIndex);

  // Extract individual issue elements
  const issueMatches = issuesXml.match(/<issue\s*>[\s\S]*?<\/issue\s*>/gi);

  if (!issueMatches) {
    return [];
  }

  const issues: SecurityIssue[] = [];

  for (const issueXml of issueMatches) {
    try {
      const titleMatch = issueXml.match(/<title\s*>([\s\S]*?)<\/title\s*>/i);
      const severityMatch = issueXml.match(
        /<severity\s*>([\s\S]*?)<\/severity\s*>/i
      );
      const descriptionMatch = issueXml.match(
        /<description\s*>([\s\S]*?)<\/description\s*>/i
      );
      const recommendationMatch = issueXml.match(
        /<recommendation\s*>([\s\S]*?)<\/recommendation\s*>/i
      );

      if (
        titleMatch &&
        severityMatch &&
        descriptionMatch &&
        recommendationMatch
      ) {
        const severity = severityMatch[1].trim();

        // Validate severity value
        if (!['Low', 'Medium', 'High', 'Critical'].includes(severity)) {
          console.warn(
            `Invalid severity value: ${severity}, defaulting to Medium`
          );
        }

        issues.push({
          title: titleMatch[1].trim(),
          severity: ['Low', 'Medium', 'High', 'Critical'].includes(severity)
            ? (severity as 'Low' | 'Medium' | 'High' | 'Critical')
            : 'Medium',
          description: descriptionMatch[1].trim(),
          recommendation: recommendationMatch[1].trim()
        });
      }
    } catch (error) {
      console.warn('Error parsing individual issue:', error);
      continue;
    }
  }

  return issues;
}

/**
 * Helper function to find the most relevant profile for a file path
 * @param filePath The file path to find a profile for
 * @param allProfiles Array of all available profiles
 * @returns The most relevant profile or undefined
 */
function findRelevantProfile(
  filePath: string,
  allProfiles: StoredProfile[]
): StoredProfile | undefined {
  if (allProfiles.length === 0) {
    return undefined;
  }

  // Check if file is in a workspace folder
  const workspaceFolder = vscode.workspace.getWorkspaceFolder(
    vscode.Uri.file(filePath)
  );
  if (!workspaceFolder) {
    return undefined;
  }

  // Find profiles within the workspace
  const relevantProfiles = allProfiles.filter((profile) => {
    const profileUri = vscode.Uri.parse(profile.workspaceFolderUri);
    const profileWorkspace = vscode.workspace.getWorkspaceFolder(profileUri);
    return (
      profileWorkspace &&
      profileWorkspace.uri.fsPath === workspaceFolder.uri.fsPath
    );
  });

  if (relevantProfiles.length === 0) {
    return undefined;
  }

  // Get relative path within workspace
  const relativePath = path.relative(workspaceFolder.uri.fsPath, filePath);

  // Find profile where file path falls under profile path
  return relevantProfiles.find((profile) => {
    const normalizedProfilePath = profile.path === '/' ? '' : profile.path;
    const profilePathParts = normalizedProfilePath.split('/').filter((p) => p);
    const filePathParts = relativePath.split(path.sep).filter((p) => p);

    // Check if file path starts with profile path parts
    return profilePathParts.every(
      (part, index) => filePathParts[index] === part
    );
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
  aiModel: string,
  apiKey: string,
  filePath?: string,
  context?: vscode.ExtensionContext,
  isGitDiff?: boolean
): Promise<SecurityIssue[]> {
  try {
    const aiClient = AIClientFactory.getClient(aiModel);
    let prompt = '';

    if (isGitDiff) {
      prompt = code;
    } else {
      prompt = await loadPrompt('common/review-changes.txt');

      if (filePath && context) {
        const profileService = new ProfileStorageService(context);
        const profile = findRelevantProfile(
          filePath,
          profileService.getAllProfiles()
        );

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
          prompt += `\n${profileContext}`;
        }
      }

      const fileName = filePath ? path.basename(filePath) : 'selected-code';
      prompt += `\n=== FILE: ${fileName} ===\n`;
      prompt += `<SELECTED_CODE>\n${code}\n</SELECTED_CODE>\n`;
      prompt += `\n=== END FILE: ${fileName} ===\n\n`;
    }

    const response = await aiClient.sendRequest(prompt, {
      apiKey,
      model: aiModel,
      temperature: 0,
      maxTokens: 2000
    });

    console.log(response);

    try {
      return parseXMLResponse(response.content.trim());
    } catch (parseError) {
      console.error('Error parsing AI response:', parseError);
      return [
        {
          title: 'Error Analyzing Code',
          severity: 'Medium',
          description: `The AI response could not be parsed: ${parseError}`,
          recommendation: 'Try again or use a different AI model.'
        }
      ];
    }
  } catch (error) {
    console.error('Error analyzing security with AI:', error);
    return [
      {
        title: 'AI Analysis Error',
        severity: 'Medium',
        description: `An error occurred while analyzing code with the AI model: ${error}`,
        recommendation:
          'Check your API key and internet connection and try again.'
      }
    ];
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
  aiModel: string,
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
                Review the following code for security issues and return XML with issues found.
                Each issue should have the following format:
                <issues>
                    <issue>
                        <title>Issue title</title>
                        <severity>Low|Medium|High|Critical</severity>
                        <description>Detailed description of the issue</description>
                        <recommendation>How to fix the issue</recommendation>
                    </issue>
                </issues>
                
                If no issues are found, return <issues></issues>.
                
                Here's the code to analyze:
                
                \`\`\`
                ${code}
                \`\`\`
                
                Please provide your response with the XML format above.
            `;

      let responseText = '';

      // Send the streaming request to the AI model
      aiClient
        .sendStreamingRequest(
          prompt,
          (chunk) => {
            responseText = chunk.content;
            progressCallback(chunk.content);

            if (chunk.isComplete) {
              try {
                const issues = parseXMLResponse(responseText);
                resolve(issues);
              } catch (parseError) {
                console.error('Error parsing AI response:', parseError);
                resolve([
                  {
                    title: 'Error Analyzing Code',
                    severity: 'Medium',
                    description: `The AI response could not be parsed: ${parseError}`,
                    recommendation: 'Try again or use a different AI model.'
                  }
                ]);
              }
            }
          },
          {
            apiKey,
            model: aiModel,
            temperature: 0.3,
            maxTokens: 2000
          }
        )
        .catch((error) => {
          console.error('Error in streaming request:', error);
          reject(error);
        });
    } catch (error) {
      console.error('Error analyzing security with AI streaming:', error);
      resolve([
        {
          title: 'AI Analysis Error',
          severity: 'Medium',
          description: `An error occurred while analyzing code with the AI model: ${error}`,
          recommendation:
            'Check your API key and internet connection and try again.'
        }
      ]);
    }
  });
}
