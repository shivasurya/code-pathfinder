import * as https from 'https';
import * as http from 'http';
import { URL } from 'url';

/**
 * Base HTTP client for making requests to AI APIs
 */
export class HttpClient {
    /**
     * Make a GET request
     * @param url The URL to request
     * @param headers Additional headers to include
     * @returns The response data
     */
    protected async get(url: string, headers: Record<string, string> = {}): Promise<any> {
        return this.request(url, 'GET', headers);
    }

    /**
     * Make a POST request
     * @param url The URL to request
     * @param body The request body
     * @param headers Additional headers to include
     * @returns The response data
     */
    protected async post(url: string, body: any, headers: Record<string, string> = {}): Promise<any> {
        return this.request(url, 'POST', headers, body);
    }

    /**
     * Make a streaming POST request
     * @param url The URL to request
     * @param body The request body
     * @param onChunk Callback for each chunk of data
     * @param headers Additional headers to include
     */
    protected async streamingPost(
        url: string, 
        body: any, 
        onChunk: (chunk: any) => void,
        onComplete: () => void,
        headers: Record<string, string> = {}
    ): Promise<void> {
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
            const req = (parsedUrl.protocol === 'https:' ? https : http).request(options, (res) => {
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
            });

            req.on('error', (error) => {
                reject(error);
            });

            req.write(JSON.stringify(body));
            req.end();
        });
    }

    /**
     * Make a generic HTTP request
     * @param url The URL to request
     * @param method The HTTP method
     * @param headers Additional headers to include
     * @param body The request body (for POST, PUT, etc.)
     * @returns The response data
     */
    private async request(url: string, method: string, headers: Record<string, string> = {}, body?: any): Promise<any> {
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
            const req = (parsedUrl.protocol === 'https:' ? https : http).request(options, (res) => {
                let data = '';

                res.on('data', (chunk) => {
                    data += chunk;
                });

                res.on('end', () => {
                    try {
                        if (res.statusCode && (res.statusCode < 200 || res.statusCode >= 300)) {
                            reject(new Error(`Request failed with status code ${res.statusCode}: ${data}`));
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
            });

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
