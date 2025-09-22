const { AIClient } = require('./ai-client');
const { HttpClient } = require('./http-client');

class GeminiClient extends HttpClient {
  constructor() {
    super();
    this.defaultModel = 'gemini-2.5-pro';
    this.maxRetries = 3;
    this.baseRetryDelay = 1000; // 1 second base delay
  }

  /**
   * Send a request to the Google Gemini API with retry logic
   * @param {string} prompt The prompt to send
   * @param {import('./ai-client').AIClientOptions} options Gemini-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<import('./ai-client').AIResponse>} The AI response
   */
  async sendRequest(prompt, options, messages) {
    if (!options?.apiKey) {
      throw new Error('Google Gemini API key is required');
    }

    const model = options.model || this.defaultModel;
    
    // Implement retry logic
    for (let attempt = 0; attempt <= this.maxRetries; attempt++) {
      try {
        const response = await this.post(
          `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent`,
          {
            contents: this._convertMessagesToContents(messages, prompt)
          },
          {
            'x-goog-api-key': options.apiKey,
            'Content-Type': 'application/json'
          }
        );

        // check if response is valid
        if (!response.candidates || response.candidates.length === 0) {
          throw new Error('No candidates found in response');
        }

        const content = response.candidates[0].content.parts
          .map((part) => part.text)
          .join('');

        return {
          content,
          model: model,
          provider: 'gemini',
          usage: response.usageMetadata
        };
      } catch (error) {
        const isLastAttempt = attempt === this.maxRetries;
        const retryInfo = this._parseRetryInfo(error);
        
        if (retryInfo.shouldRetry && !isLastAttempt) {
          console.log(`🔄 Rate limit hit (attempt ${attempt + 1}/${this.maxRetries + 1}). Retrying in ${retryInfo.delay}ms...`);
          console.log(`📊 Quota: ${retryInfo.quotaMetric} (limit: ${retryInfo.quotaLimit})`);
          
          await this._sleep(retryInfo.delay);
          continue;
        }
        
        // If it's not a retryable error or we've exhausted retries, throw the error
        throw error;
      }
    }
  }

  /**
   * Convert messages array to Gemini contents format
   * @param {import('./ai-client').AIMessage[]} [messages] Messages array
   * @param {string} prompt Fallback prompt if no messages
   * @returns {any[]} Gemini contents array
   */
  _convertMessagesToContents(messages, prompt) {
    if (!messages) {
      return [
        {
          parts: [
            {
              text: prompt
            }
          ]
        }
      ];
    }

    return messages.map(message => ({
      role: message.role === 'assistant' ? 'model' : 'user',
      parts: [
        {
          text: message.content
        }
      ]
    }));
  }

  /**
   * Send a streaming request to the Google Gemini API
   * @param {string} prompt The prompt to send
   * @param {function(import('./ai-client').AIResponseChunk): void} callback Callback function for each chunk
   * @param {import('./ai-client').AIClientOptions} options Gemini-specific options
   * @param {import('./ai-client').AIMessage[]} [messages] Optional messages array for conversation context
   * @returns {Promise<void>}
   */
  async sendStreamingRequest(prompt, callback, options, messages) {
    // throw not implemented error
    throw new Error('sendStreamingRequest not implemented for GeminiClient');
  }

  /**
   * Parse multiple JSON objects from a stream chunk
   * @param {string} chunk Raw stream chunk that may contain multiple JSON objects
   * @returns {any[]} Array of parsed JSON objects
   */
  parseStreamChunk(chunk) {
    // throw not implemented error
    throw new Error('parseStreamChunk not implemented for GeminiClient');
  }

  /**
   * Parse retry information from Gemini API error response
   * @param {Error} error The error object from the API request
   * @returns {Object} Retry information including delay and whether to retry
   */
  _parseRetryInfo(error) {
    const defaultRetry = {
      shouldRetry: false,
      delay: this.baseRetryDelay,
      quotaMetric: 'unknown',
      quotaLimit: 'unknown'
    };

    try {
      // Check if it's a rate limit error (429)
      if (!error.message.includes('status code 429')) {
        return defaultRetry;
      }

      // Extract JSON from error message
      const jsonMatch = error.message.match(/status code 429: ({.*})/s);
      if (!jsonMatch) {
        return defaultRetry;
      }

      const errorData = JSON.parse(jsonMatch[1]);
      
      // Check if it's a quota exceeded error
      if (errorData.error?.code !== 429 || errorData.error?.status !== 'RESOURCE_EXHAUSTED') {
        return defaultRetry;
      }

      let retryDelay = this.baseRetryDelay;
      let quotaMetric = 'unknown';
      let quotaLimit = 'unknown';

      // Parse retry delay from RetryInfo
      if (errorData.error.details) {
        for (const detail of errorData.error.details) {
          if (detail['@type'] === 'type.googleapis.com/google.rpc.RetryInfo' && detail.retryDelay) {
            // Parse delay string like "3s" or "3.679304258s"
            const delayMatch = detail.retryDelay.match(/^(\d+(?:\.\d+)?)s$/);
            if (delayMatch) {
              retryDelay = Math.ceil(parseFloat(delayMatch[1]) * 1000); // Convert to milliseconds
            }
          }
          
          // Parse quota information from QuotaFailure
          if (detail['@type'] === 'type.googleapis.com/google.rpc.QuotaFailure' && detail.violations) {
            const violation = detail.violations[0];
            if (violation) {
              quotaMetric = violation.quotaMetric || 'unknown';
              quotaLimit = violation.quotaValue || 'unknown';
            }
          }
        }
      }

      return {
        shouldRetry: true,
        delay: retryDelay,
        quotaMetric,
        quotaLimit
      };
    } catch (parseError) {
      console.warn('⚠️  Failed to parse retry info from error:', parseError.message);
      return defaultRetry;
    }
  }

  /**
   * Sleep for the specified number of milliseconds
   * @param {number} ms Milliseconds to sleep
   * @returns {Promise<void>}
   */
  _sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}

module.exports = {
  GeminiClient
};
