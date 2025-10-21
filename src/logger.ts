/**
 * Logging system with file and console output
 */

import { writeFileSync, appendFileSync, existsSync, mkdirSync } from 'fs';
import { join } from 'path';
import type { LogEntry, MessageResult, ExecutionReport } from './types';
import { config } from '../config';

class Logger {
  private logFile: string;
  private errorFile: string;
  private successFile: string;
  private logEntries: LogEntry[] = [];

  constructor() {
    // Create logs directory if it doesn't exist
    if (!existsSync(config.logging.logDirectory)) {
      mkdirSync(config.logging.logDirectory, { recursive: true });
    }

    const timestamp = new Date().toISOString().split('T')[0];
    this.logFile = join(config.logging.logDirectory, `app-${timestamp}.log`);
    this.errorFile = join(config.logging.logDirectory, `errors-${timestamp}.log`);
    this.successFile = join(config.logging.logDirectory, `success-${timestamp}.log`);
  }

  /**
   * Log a message
   */
  log(level: LogEntry['level'], message: string, data?: any): void {
    if (!config.logging.enabled) return;

    const entry: LogEntry = {
      timestamp: new Date(),
      level,
      message,
      data,
    };

    this.logEntries.push(entry);

    // Console output with colors
    if (config.logging.logToConsole) {
      this.logToConsole(entry);
    }

    // File output
    if (config.logging.logToFile) {
      this.logToFile(entry);
    }
  }

  /**
   * Log to console with colors
   */
  private logToConsole(entry: LogEntry): void {
    const timestamp = entry.timestamp.toISOString();
    const colors = {
      info: '\x1b[36m',    // Cyan
      success: '\x1b[32m', // Green
      warning: '\x1b[33m', // Yellow
      error: '\x1b[31m',   // Red
      debug: '\x1b[90m',   // Gray
    };
    const reset = '\x1b[0m';

    const color = colors[entry.level] || reset;
    const levelStr = entry.level.toUpperCase().padEnd(7);
    
    console.log(`${color}[${timestamp}] ${levelStr}${reset} ${entry.message}`);
    
    if (entry.data && config.logging.logLevel === 'debug') {
      console.log(color + JSON.stringify(entry.data, null, 2) + reset);
    }
  }

  /**
   * Log to file
   */
  private logToFile(entry: LogEntry): void {
    const timestamp = entry.timestamp.toISOString();
    const logLine = `[${timestamp}] [${entry.level.toUpperCase()}] ${entry.message}\n`;
    
    try {
      appendFileSync(this.logFile, logLine);

      // Also log to error file if it's an error
      if (entry.level === 'error') {
        const errorDetail = entry.data ? `\nData: ${JSON.stringify(entry.data, null, 2)}\n` : '';
        appendFileSync(this.errorFile, logLine + errorDetail);
      }
    } catch (error) {
      console.error('Failed to write to log file:', error);
    }
  }

  /**
   * Log info message
   */
  info(message: string, data?: any): void {
    this.log('info', message, data);
  }

  /**
   * Log success message
   */
  success(message: string, data?: any): void {
    this.log('success', message, data);
  }

  /**
   * Log warning message
   */
  warning(message: string, data?: any): void {
    this.log('warning', message, data);
  }

  /**
   * Log error message
   */
  error(message: string, data?: any): void {
    this.log('error', message, data);
  }

  /**
   * Log debug message
   */
  debug(message: string, data?: any): void {
    if (config.logging.logLevel === 'debug') {
      this.log('debug', message, data);
    }
  }

  /**
   * Log message result
   */
  logMessageResult(result: MessageResult): void {
    if (result.success) {
      this.success(
        `Message sent to ${result.customer.CustomerName} (${result.customer.formattedPhone})`,
        { retryCount: result.retryCount }
      );
      
      // Log to success file
      const successLine = `${result.timestamp.toISOString()},${result.customer.Code},${result.customer.CustomerName},${result.customer.formattedPhone},SUCCESS\n`;
      try {
        appendFileSync(this.successFile, successLine);
      } catch (error) {
        this.error('Failed to write to success file', error);
      }
    } else {
      this.error(
        `Failed to send message to ${result.customer.CustomerName} (${result.customer.formattedPhone}): ${result.error}`,
        { retryCount: result.retryCount }
      );
    }
  }

  /**
   * Generate execution report
   */
  generateReport(report: ExecutionReport): void {
    const reportFile = join(
      config.logging.logDirectory,
      `report-${new Date().toISOString().split('T')[0]}.json`
    );

    try {
      writeFileSync(reportFile, JSON.stringify(report, null, 2));
      this.info(`Execution report saved to ${reportFile}`);
    } catch (error) {
      this.error('Failed to save execution report', error);
    }

    // Print summary to console
    this.printSummary(report);
  }

  /**
   * Print execution summary
   */
  private printSummary(report: ExecutionReport): void {
    const duration = Math.floor(report.duration / 1000);
    const minutes = Math.floor(duration / 60);
    const seconds = duration % 60;

    console.log('\n' + '='.repeat(60));
    console.log('EXECUTION SUMMARY');
    console.log('='.repeat(60));
    console.log(`Start Time:         ${report.startTime.toLocaleString()}`);
    console.log(`End Time:           ${report.endTime.toLocaleString()}`);
    console.log(`Duration:           ${minutes}m ${seconds}s`);
    console.log(`Total Customers:    ${report.totalCustomers}`);
    console.log(`Successful Sends:   ${report.successfulSends} (${report.successRate.toFixed(2)}%)`);
    console.log(`Failed Sends:       ${report.failedSends}`);
    console.log(`Skipped Customers:  ${report.skippedCustomers}`);
    console.log(`Average Delay:      ${(report.averageDelay / 1000).toFixed(2)}s`);
    
    if (report.errors.length > 0) {
      console.log(`\nErrors (${report.errors.length}):`);
      report.errors.slice(0, 5).forEach(err => {
        console.log(`  - ${err.customer}: ${err.error}`);
      });
      if (report.errors.length > 5) {
        console.log(`  ... and ${report.errors.length - 5} more errors`);
      }
    }
    
    console.log('='.repeat(60) + '\n');
  }

  /**
   * Display progress
   */
  displayProgress(current: number, total: number, customerName: string): void {
    const percentage = ((current / total) * 100).toFixed(1);
    const bar = this.createProgressBar(current, total, 30);
    
    process.stdout.write(`\r${bar} ${percentage}% (${current}/${total}) - Processing: ${customerName.substring(0, 30).padEnd(30)}`);
  }

  /**
   * Create progress bar
   */
  private createProgressBar(current: number, total: number, length: number): string {
    const filled = Math.floor((current / total) * length);
    const empty = length - filled;
    return '[' + '█'.repeat(filled) + '░'.repeat(empty) + ']';
  }

  /**
   * Clear progress line
   */
  clearProgress(): void {
    process.stdout.write('\r' + ' '.repeat(120) + '\r');
  }
}

// Export singleton instance
export const logger = new Logger();
