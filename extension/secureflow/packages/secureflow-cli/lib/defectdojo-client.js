/**
 * DefectDojo API Client
 * Handles communication with DefectDojo API for submitting security findings
 * 
 * API Documentation: https://docs.defectdojo.com/en/api_v2_docs/
 */

const https = require('https');
const http = require('http');
const { URL } = require('url');

class DefectDojoClient {
  constructor(options = {}) {
    this.baseUrl = options.url;
    this.token = options.token;
    this.productId = options.productId;
    this.engagementId = options.engagementId; // Can be null, will be created if needed
    this.testTitle = options.testTitle || 'SecureFlow Scan';
    
    if (!this.baseUrl) {
      throw new Error('DefectDojo URL is required');
    }
    if (!this.token) {
      throw new Error('DefectDojo API token is required');
    }
    if (!this.productId) {
      throw new Error('DefectDojo product ID is required');
    }

    // Normalize base URL
    this.baseUrl = this.baseUrl.replace(/\/$/, '');
  }

  /**
   * Submit findings to DefectDojo via Generic Findings Import
   * @param {Object} findings - DefectDojo formatted findings object
   * @returns {Promise<Object>} API response
   */
  async submitFindings(findings) {
    try {
      // First, create or get a test for this scan
      const test = await this._createTest();
      
      // Then submit the findings to the test
      const result = await this._importFindings(test.id, findings);
      
      return {
        success: true,
        testId: test.id,
        testUrl: `${this.baseUrl}/test/${test.id}`,
        findingsImported: result.findings_count || findings.findings.length,
        response: result
      };
    } catch (error) {
      throw new Error(`DefectDojo API submission failed: ${error.message}`);
    }
  }

  /**
   * Create a new test in DefectDojo for this scan
   * @returns {Promise<Object>} Test object
   */
  async _createTest() {
    const testData = {
      title: this.testTitle,
      engagement: parseInt(this.engagementId),
      test_type: await this._getGenericTestTypeId(),
      target_start: new Date().toISOString().split('T')[0],
      target_end: new Date().toISOString().split('T')[0],
      environment: 1, // Default environment
      scan_type: 'Static',
      description: `Automated security scan performed by SecureFlow CLI at ${new Date().toISOString()}`
    };

    const response = await this._makeRequest('POST', '/api/v2/tests/', testData);
    return response;
  }

  /**
   * Get the test type ID for Generic scanner
   * @returns {Promise<number>} Test type ID
   */
  async _getGenericTestTypeId() {
    try {
      const response = await this._makeRequest('GET', '/api/v2/test_types/?name=Generic%20Findings%20Import');
      
      if (response.results && response.results.length > 0) {
        return response.results[0].id;
      }
      
      // Fallback: try to find any generic test type
      const fallbackResponse = await this._makeRequest('GET', '/api/v2/test_types/?limit=100');
      const genericType = fallbackResponse.results?.find(type => 
        type.name.toLowerCase().includes('generic') || 
        type.name.toLowerCase().includes('import')
      );
      
      if (genericType) {
        return genericType.id;
      }
      
      throw new Error('Could not find Generic Findings Import test type');
    } catch (error) {
      throw new Error(`Failed to get test type: ${error.message}`);
    }
  }

  /**
   * Import findings into a DefectDojo test
   * @param {number} testId - Test ID to import findings into
   * @param {Object} findings - DefectDojo formatted findings
   * @returns {Promise<Object>} Import response
   */
  async _importFindings(testId, findings) {
    const findingsJson = JSON.stringify(findings);
    
    // Get product and engagement names for the import
    const product = await this._makeRequest('GET', `/api/v2/products/${this.productId}/`);
    const engagement = await this._makeRequest('GET', `/api/v2/engagements/${this.engagementId}/`);
    
    // Create multipart form data
    const boundary = `----formdata-secureflow-${Date.now()}`;
    const formData = this._createMultipartFormData(boundary, {
      test: testId.toString(),
      scan_type: 'Generic Findings Import',
      product_name: product.name,
      engagement_name: engagement.name,
      active: 'true',
      verified: 'false',
      minimum_severity: 'Info',
      close_old_findings: 'false',
      push_to_jira: 'false',
      create_finding_groups_for_all_findings: 'true',
      scan_date: new Date().toISOString().split('T')[0]
    }, {
      fieldName: 'file',
      fileName: 'secureflow-findings.json',
      content: findingsJson,
      contentType: 'application/json'
    });

    const response = await this._makeMultipartRequest('POST', '/api/v2/import-scan/', formData, boundary);
    return response;
  }

  /**
   * Create multipart form data
   * @param {string} boundary - Form boundary
   * @param {Object} fields - Form fields
   * @param {Object} file - File data
   * @returns {Buffer} Form data buffer
   */
  _createMultipartFormData(boundary, fields, file) {
    let formData = '';
    
    // Add form fields
    for (const [key, value] of Object.entries(fields)) {
      formData += `--${boundary}\r\n`;
      formData += `Content-Disposition: form-data; name="${key}"\r\n\r\n`;
      formData += `${value}\r\n`;
    }
    
    // Add file
    formData += `--${boundary}\r\n`;
    formData += `Content-Disposition: form-data; name="${file.fieldName}"; filename="${file.fileName}"\r\n`;
    formData += `Content-Type: ${file.contentType}\r\n\r\n`;
    formData += file.content;
    formData += `\r\n--${boundary}--\r\n`;
    
    return Buffer.from(formData);
  }

  /**
   * Make multipart HTTP request to DefectDojo API
   * @param {string} method - HTTP method
   * @param {string} path - API path
   * @param {Buffer} formData - Multipart form data
   * @param {string} boundary - Form boundary
   * @returns {Promise<Object>} Response data
   */
  async _makeMultipartRequest(method, path, formData, boundary) {
    return new Promise((resolve, reject) => {
      const url = new URL(path, this.baseUrl);
      const isHttps = url.protocol === 'https:';
      const httpModule = isHttps ? https : http;
      
      const options = {
        hostname: url.hostname,
        port: url.port || (isHttps ? 443 : 80),
        path: url.pathname + url.search,
        method: method,
        headers: {
          'Authorization': `Token ${this.token}`,
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
          'Content-Length': formData.length,
          'Accept': 'application/json'
        }
      };

      const req = httpModule.request(options, (res) => {
        let responseData = '';
        
        res.on('data', (chunk) => {
          responseData += chunk;
        });
        
        res.on('end', () => {
          try {
            const parsedData = responseData ? JSON.parse(responseData) : {};
            
            if (res.statusCode >= 200 && res.statusCode < 300) {
              resolve(parsedData);
            } else {
              const errorMessage = parsedData.detail || 
                                 parsedData.message || 
                                 parsedData.error ||
                                 parsedData.non_field_errors?.[0] ||
                                 `HTTP ${res.statusCode}: ${res.statusMessage}`;
              reject(new Error(errorMessage));
            }
          } catch (parseError) {
            reject(new Error(`Failed to parse response: ${parseError.message}. Response: ${responseData}`));
          }
        });
      });

      req.on('error', (error) => {
        reject(new Error(`Request failed: ${error.message}`));
      });

      req.on('timeout', () => {
        req.destroy();
        reject(new Error('Request timeout'));
      });

      // Set timeout
      req.setTimeout(60000); // Increase timeout for file upload

      req.write(formData);
      req.end();
    });
  }

  /**
   * Make HTTP request to DefectDojo API
   * @param {string} method - HTTP method
   * @param {string} path - API path
   * @param {Object} data - Request data
   * @returns {Promise<Object>} Response data
   */
  async _makeRequest(method, path, data = null) {
    return new Promise((resolve, reject) => {
      const url = new URL(path, this.baseUrl);
      const isHttps = url.protocol === 'https:';
      const httpModule = isHttps ? https : http;
      
      const options = {
        hostname: url.hostname,
        port: url.port || (isHttps ? 443 : 80),
        path: url.pathname + url.search,
        method: method,
        headers: {
          'Authorization': `Token ${this.token}`,
          'Content-Type': 'application/json',
          'Accept': 'application/json'
        }
      };

      const postData = data ? JSON.stringify(data) : null;
      if (postData) {
        options.headers['Content-Length'] = Buffer.byteLength(postData);
      }

      const req = httpModule.request(options, (res) => {
        let responseData = '';
        
        res.on('data', (chunk) => {
          responseData += chunk;
        });
        
        res.on('end', () => {
          try {
            const parsedData = responseData ? JSON.parse(responseData) : {};
            
            if (res.statusCode >= 200 && res.statusCode < 300) {
              resolve(parsedData);
            } else {
              const errorMessage = parsedData.detail || 
                                 parsedData.message || 
                                 parsedData.error ||
                                 `HTTP ${res.statusCode}: ${res.statusMessage}`;
              reject(new Error(errorMessage));
            }
          } catch (parseError) {
            reject(new Error(`Failed to parse response: ${parseError.message}`));
          }
        });
      });

      req.on('error', (error) => {
        reject(new Error(`Request failed: ${error.message}`));
      });

      req.on('timeout', () => {
        req.destroy();
        reject(new Error('Request timeout'));
      });

      // Set timeout
      req.setTimeout(30000);

      if (postData) {
        req.write(postData);
      }
      
      req.end();
    });
  }

  /**
   * Test connection to DefectDojo API
   * @returns {Promise<Object>} Connection test result
   */
  async testConnection() {
    try {
      const response = await this._makeRequest('GET', '/api/v2/users/');
      return {
        success: true,
        message: 'Successfully connected to DefectDojo API',
        userCount: response.count || 0
      };
    } catch (error) {
      return {
        success: false,
        message: `Connection failed: ${error.message}`
      };
    }
  }

  /**
   * Validate DefectDojo configuration and create engagement if needed
   * @returns {Promise<Object>} Validation result
   */
  async validateConfiguration() {
    const errors = [];
    let engagementCreated = false;
    
    try {
      // Test basic connection
      const connectionTest = await this.testConnection();
      if (!connectionTest.success) {
        errors.push(`API connection failed: ${connectionTest.message}`);
        return { valid: false, errors };
      }

      // Validate product exists
      let product;
      try {
        product = await this._makeRequest('GET', `/api/v2/products/${this.productId}/`);
      } catch (error) {
        errors.push(`Product ID ${this.productId} not found or not accessible`);
        return { valid: false, errors };
      }

      // Validate engagement exists, create if not found or not provided
      if (this.engagementId) {
        try {
          await this._makeRequest('GET', `/api/v2/engagements/${this.engagementId}/`);
        } catch (error) {
          // Engagement not found, try to create it
          try {
            const newEngagement = await this._createEngagement(product);
            this.engagementId = newEngagement.id.toString();
            engagementCreated = true;
          } catch (createError) {
            errors.push(`Engagement ID ${this.engagementId} not found and could not create new engagement: ${createError.message}`);
          }
        }
      } else {
        // No engagement ID provided, create a new one
        try {
          const newEngagement = await this._createEngagement(product);
          this.engagementId = newEngagement.id.toString();
          engagementCreated = true;
        } catch (createError) {
          errors.push(`No engagement ID provided and could not create new engagement: ${createError.message}`);
        }
      }

      // Check if Generic Findings Import test type is available
      try {
        await this._getGenericTestTypeId();
      } catch (error) {
        errors.push(`Generic Findings Import test type not available: ${error.message}`);
      }

      return {
        valid: errors.length === 0,
        errors: errors,
        engagementCreated: engagementCreated,
        engagementId: this.engagementId
      };
    } catch (error) {
      return {
        valid: false,
        errors: [`Configuration validation failed: ${error.message}`]
      };
    }
  }

  /**
   * Create a new engagement in DefectDojo
   * @param {Object} product - Product object from DefectDojo
   * @returns {Promise<Object>} Created engagement object
   */
  async _createEngagement(product) {
    const engagementData = {
      name: `SecureFlow Engagement - ${new Date().toISOString().split('T')[0]}`,
      description: `Automated engagement created by SecureFlow CLI for security scanning`,
      product: parseInt(this.productId),
      target_start: new Date().toISOString().split('T')[0],
      target_end: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0], // 30 days from now
      reason: 'Automated Security Scanning',
      updated: new Date().toISOString(),
      active: true,
      tracker: `${this.baseUrl}/engagement/`,
      test_strategy: `${this.baseUrl}/test_strategy/`,
      threat_model: true,
      api_test: true,
      pen_test: false,
      check_list: true,
      status: 'In Progress',
      engagement_type: 'CI/CD',
      build_id: `secureflow-${Date.now()}`,
      commit_hash: '',
      branch_tag: 'main',
      source_code_management_uri: '',
      deduplication_on_engagement: true
    };

    const response = await this._makeRequest('POST', '/api/v2/engagements/', engagementData);
    return response;
  }
}

module.exports = { DefectDojoClient };
