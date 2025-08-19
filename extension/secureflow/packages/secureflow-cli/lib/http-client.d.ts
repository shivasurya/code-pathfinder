/**
 * Base HTTP client for making requests to AI APIs
 */
export declare class HttpClient {
  /**
   * Make a GET request
   * @param url The URL to request
   * @param headers Additional headers to include
   * @returns The response data
   */
  protected get(url: string, headers?: Record<string, string>): Promise<any>;

  /**
   * Make a POST request
   * @param url The URL to request
   * @param body The request body
   * @param headers Additional headers to include
   * @returns The response data
   */
  protected post(url: string, body: any, headers?: Record<string, string>): Promise<any>;

  /**
   * Make a streaming POST request
   * @param url The URL to request
   * @param body The request body
   * @param onChunk Callback for each chunk of data
   * @param onComplete Callback when request completes
   * @param headers Additional headers to include
   */
  protected streamingPost(
    url: string,
    body: any,
    onChunk: (chunk: any) => void,
    onComplete: () => void,
    headers?: Record<string, string>
  ): Promise<void>;
}
