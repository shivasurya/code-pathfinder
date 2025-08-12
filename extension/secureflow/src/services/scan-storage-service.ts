import * as vscode from 'vscode';
import { ScanResult, ScanStorageData } from '../models/scan-result';
import { SecurityIssue } from '../models/security-issue';

/**
 * Service for managing scan results storage
 */
export class ScanStorageService {
  private static readonly STORE_KEY = 'secureflow.scanStore';
  private context: vscode.ExtensionContext;
  private data: ScanStorageData;

  /**
   * Create a new ScanStorageService
   */
  constructor(context: vscode.ExtensionContext) {
    this.context = context;
    this.data = this.loadData();
  }

  /**
   * Load scan store data from storage
   */
  private loadData(): ScanStorageData {
    const data = this.context.globalState.get<ScanStorageData>(
      ScanStorageService.STORE_KEY
    );
    return (
      data || {
        scans: {},
        nextScanNumber: 1,
        version: 1
      }
    );
  }

  /**
   * Save scan store data to storage
   */
  private async saveData(): Promise<void> {
    await this.context.globalState.update(
      ScanStorageService.STORE_KEY,
      this.data
    );
  }

  /**
   * Save a new scan result
   *
   * @param issues Array of security issues found
   * @param summary Summary of the scan
   * @param reviewContent The consolidated review content that was analyzed
   * @param fileCount Number of files analyzed
   * @param model AI model used for analysis
   * @returns The saved scan result with assigned scan number
   */
  public async saveScan(
    issues: Array<{
      issue: SecurityIssue;
      filePath: string;
      startLine: number;
    }>,
    summary: string,
    reviewContent: string,
    fileCount: number,
    model: string
  ): Promise<ScanResult> {
    const scanNumber = this.data.nextScanNumber;
    const timestamp = Date.now();
    const timestampFormatted = new Date(timestamp).toLocaleString();

    const scanResult: ScanResult = {
      scanNumber,
      timestamp,
      timestampFormatted,
      issues,
      summary,
      reviewContent,
      fileCount,
      model
    };

    // Save the scan
    this.data.scans[scanNumber] = scanResult;
    this.data.nextScanNumber++;

    await this.saveData();
    return scanResult;
  }

  /**
   * Retrieve a scan by its scan number
   *
   * @param scanNumber The scan number to retrieve
   * @returns The scan result or undefined if not found
   */
  public getScanByNumber(scanNumber: number): ScanResult | undefined {
    return this.data.scans[scanNumber];
  }

  /**
   * Get all saved scans
   *
   * @returns Array of all scan results, sorted by scan number (newest first)
   */
  public getAllScans(): ScanResult[] {
    return Object.values(this.data.scans).sort(
      (a, b) => b.scanNumber - a.scanNumber
    );
  }

  /**
   * Get the latest scan
   *
   * @returns The most recent scan result or undefined if no scans exist
   */
  public getLatestScan(): ScanResult | undefined {
    const scans = this.getAllScans();
    return scans.length > 0 ? scans[0] : undefined;
  }

  /**
   * Delete a scan by its scan number
   *
   * @param scanNumber The scan number to delete
   * @returns True if the scan was deleted, false if it didn't exist
   */
  public async deleteScan(scanNumber: number): Promise<boolean> {
    if (this.data.scans[scanNumber]) {
      delete this.data.scans[scanNumber];
      await this.saveData();
      return true;
    }
    return false;
  }

  /**
   * Clear all saved scans
   */
  public async clearAllScans(): Promise<void> {
    this.data.scans = {};
    this.data.nextScanNumber = 1;
    await this.saveData();
  }

  /**
   * Get scan statistics
   *
   * @returns Object with scan statistics
   */
  public getStats(): {
    totalScans: number;
    nextScanNumber: number;
    totalIssues: number;
    latestScanTimestamp?: string;
  } {
    const scans = Object.values(this.data.scans);
    const totalIssues = scans.reduce(
      (sum, scan) => sum + scan.issues.length,
      0
    );
    const latestScan = this.getLatestScan();

    return {
      totalScans: scans.length,
      nextScanNumber: this.data.nextScanNumber,
      totalIssues,
      latestScanTimestamp: latestScan?.timestampFormatted
    };
  }
}
