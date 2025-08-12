import { SecurityIssue } from './security-issue';

/**
 * Represents a saved security scan result
 */
export interface ScanResult {
  /**
   * Unique scan number (running number starting from 1)
   */
  scanNumber: number;

  /**
   * Timestamp when the scan was performed
   */
  timestamp: number;

  /**
   * Human-readable timestamp
   */
  timestampFormatted: string;

  /**
   * Array of security issues found in the scan
   */
  issues: Array<{
    issue: SecurityIssue;
    filePath: string;
    startLine: number;
  }>;

  /**
   * Summary of the scan
   */
  summary: string;

  /**
   * The consolidated review content that was analyzed
   */
  reviewContent: string;

  /**
   * Number of files analyzed
   */
  fileCount: number;

  /**
   * AI model used for the analysis
   */
  model: string;
}

/**
 * Schema for the scan storage data
 */
export interface ScanStorageData {
  /**
   * Map of scan numbers to scan results
   */
  scans: { [scanNumber: number]: ScanResult };

  /**
   * The next scan number to use
   */
  nextScanNumber: number;

  /**
   * Version of the storage schema, for future migrations
   */
  version: number;
}
