const https = require('https');
const http = require('http');
const { URL } = require('url');

/**
 * Base HTTP client for making requests to AI APIs
 */
class HttpClient {
  /**
   * Make a GET request
   * @param {string} url The URL to request
   * @param {Record<string, string>} headers Additional headers to include
   * @returns {Promise<any>} The response data
   */
  async get(url, headers = {}) {
    return this.request(url, 'GET', headers);
  }

  /**
   * Make a POST request
   * @param {string} url The URL to request
   * @param {any} body The request body
   * @param {Record<string, string>} headers Additional headers to include
   * @returns {Promise<any>} The response data
   */
  async post(url, body, headers = {}) {
    return this.request(url, 'POST', headers, body);
  }

  /**
   * Make a streaming POST request
   * @param {string} url The URL to request
   * @param {any} body The request body
   * @param {function(any): void} onChunk Callback for each chunk of data
   * @param {function(): void} onComplete Callback when request completes
   * @param {Record<string, string>} headers Additional headers to include
   * @returns {Promise<void>}
   */
  async streamingPost(url, body, onChunk, onComplete, headers = {}) {
    const parsedUrl = new URL(url);
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
      path: parsedUrl.pathname + parsedUrl.search,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...headers
      }
    };

    return new Promise((resolve, reject) => {
      const req = (parsedUrl.protocol === 'https:' ? https : http).request(
        options,
        (res) => {
          res.on('data', (chunk) => {
            try {
              onChunk(chunk.toString());
            } catch (error) {
              console.error('Error processing chunk:', error);
            }
          });

          res.on('end', () => {
            onComplete();
            resolve();
          });

          res.on('error', (error) => {
            reject(error);
          });
        }
      );

      req.on('error', (error) => {
        reject(error);
      });

      req.write(JSON.stringify(body));
      req.end();
    });
  }

  /**
   * Make a generic HTTP request
   * @param {string} url The URL to request
   * @param {string} method The HTTP method
   * @param {Record<string, string>} headers Additional headers to include
   * @param {any} body The request body (for POST, PUT, etc.)
   * @returns {Promise<any>} The response data
   */
  async request(url, method, headers = {}, body) {
    const parsedUrl = new URL(url);
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
      path: parsedUrl.pathname + parsedUrl.search,
      method,
      headers: {
        'Content-Type': 'application/json',
        ...headers
      }
    };

    return new Promise((resolve, reject) => {
      const req = (parsedUrl.protocol === 'https:' ? https : http).request(
        options,
        (res) => {
          let data = '';

          res.on('data', (chunk) => {
            data += chunk;
          });

          res.on('end', () => {
            try {
              if (
                res.statusCode &&
                (res.statusCode < 200 || res.statusCode >= 300)
              ) {
                reject(
                  new Error(
                    `Request failed with status code ${res.statusCode}: ${data}`
                  )
                );
                return;
              }

              try {
                const jsonData = JSON.parse(data);
                resolve(jsonData);
              } catch (e) {
                // If it's not JSON, return the raw data
                resolve(data);
              }
            } catch (e) {
              reject(e);
            }
          });

          res.on('error', (error) => {
            reject(error);
          });
        }
      );

      req.on('error', (error) => {
        reject(error);
      });

      if (body) {
        req.write(JSON.stringify(body));
      }

      req.end();
    });
  }
}

module.exports = {
  HttpClient
};
