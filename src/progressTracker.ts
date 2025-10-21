/**
 * Progress tracking and state management
 */

import { writeFileSync, readFileSync, existsSync } from 'fs';
import type { ProgressState, ExecutionReport, MessageResult } from './types';
import { logger } from './logger';

class ProgressTracker {
  private state: ProgressState;
  private messageResults: MessageResult[] = [];
  private delays: number[] = [];
  private progressFile: string = './data/progress.json';

  constructor() {
    this.state = {
      totalCustomers: 0,
      processedCustomers: 0,
      successfulSends: 0,
      failedSends: 0,
      skippedCustomers: 0,
      startTime: new Date(),
      lastProcessedIndex: -1,
      currentBatch: 0,
    };
  }

  /**
   * Initialize progress tracking
   */
  initialize(totalCustomers: number): void {
    this.state.totalCustomers = totalCustomers;
    this.state.startTime = new Date();
    this.saveProgress();
  }

  /**
   * Load progress from file (for resume functionality)
   */
  loadProgress(): ProgressState | null {
    try {
      if (existsSync(this.progressFile)) {
        const data = readFileSync(this.progressFile, 'utf-8');
        const savedState = JSON.parse(data);
        savedState.startTime = new Date(savedState.startTime);
        logger.info('Loaded previous progress state');
        return savedState;
      }
    } catch (error) {
      logger.warning('Failed to load progress state', error);
    }
    return null;
  }

  /**
   * Save progress to file
   */
  saveProgress(): void {
    try {
      // Ensure data directory exists
      const fs = require('fs');
      const path = require('path');
      const dir = path.dirname(this.progressFile);
      
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }

      writeFileSync(this.progressFile, JSON.stringify(this.state, null, 2));
    } catch (error) {
      logger.error('Failed to save progress', error);
    }
  }

  /**
   * Record message result
   */
  recordResult(result: MessageResult, delay: number): void {
    this.messageResults.push(result);
    this.delays.push(delay);
    
    this.state.processedCustomers++;
    
    if (result.success) {
      this.state.successfulSends++;
    } else {
      this.state.failedSends++;
    }

    this.state.lastProcessedIndex = this.state.processedCustomers - 1;
    
    // Save progress every 10 messages
    if (this.state.processedCustomers % 10 === 0) {
      this.saveProgress();
    }
  }

  /**
   * Record skipped customer
   */
  recordSkipped(): void {
    this.state.skippedCustomers++;
    this.state.processedCustomers++;
  }

  /**
   * Increment batch counter
   */
  incrementBatch(): void {
    this.state.currentBatch++;
  }

  /**
   * Get current state
   */
  getState(): ProgressState {
    return { ...this.state };
  }

  /**
   * Get success rate
   */
  getSuccessRate(): number {
    const attempted = this.state.successfulSends + this.state.failedSends;
    if (attempted === 0) return 0;
    return (this.state.successfulSends / attempted) * 100;
  }

  /**
   * Get average delay
   */
  getAverageDelay(): number {
    if (this.delays.length === 0) return 0;
    return this.delays.reduce((a, b) => a + b, 0) / this.delays.length;
  }

  /**
   * Calculate ETA
   */
  calculateETA(): { minutes: number; seconds: number } {
    const remaining = this.state.totalCustomers - this.state.processedCustomers;
    if (remaining === 0 || this.delays.length === 0) {
      return { minutes: 0, seconds: 0 };
    }

    const avgDelay = this.getAverageDelay();
    const etaMs = remaining * avgDelay;
    const etaSeconds = Math.ceil(etaMs / 1000);
    
    return {
      minutes: Math.floor(etaSeconds / 60),
      seconds: etaSeconds % 60,
    };
  }

  /**
   * Generate execution report
   */
  generateReport(): ExecutionReport {
    const endTime = new Date();
    const duration = endTime.getTime() - this.state.startTime.getTime();
    
    const errors = this.messageResults
      .filter(r => !r.success)
      .map(r => ({
        customer: r.customer.CustomerName,
        error: r.error || 'Unknown error',
        timestamp: r.timestamp,
      }));

    return {
      startTime: this.state.startTime,
      endTime,
      duration,
      totalCustomers: this.state.totalCustomers,
      successfulSends: this.state.successfulSends,
      failedSends: this.state.failedSends,
      skippedCustomers: this.state.skippedCustomers,
      successRate: this.getSuccessRate(),
      averageDelay: this.getAverageDelay(),
      errors,
    };
  }

  /**
   * Get failed customers
   */
  getFailedCustomers() {
    return this.messageResults
      .filter(r => !r.success)
      .map(r => r.customer);
  }

  /**
   * Display real-time stats
   */
  displayStats(): void {
    const eta = this.calculateETA();
    const successRate = this.getSuccessRate();
    
    console.log('\n' + '─'.repeat(60));
    console.log('CURRENT STATISTICS');
    console.log('─'.repeat(60));
    console.log(`Processed:     ${this.state.processedCustomers}/${this.state.totalCustomers}`);
    console.log(`Successful:    ${this.state.successfulSends}`);
    console.log(`Failed:        ${this.state.failedSends}`);
    console.log(`Skipped:       ${this.state.skippedCustomers}`);
    console.log(`Success Rate:  ${successRate.toFixed(2)}%`);
    console.log(`Current Batch: ${this.state.currentBatch}`);
    console.log(`ETA:           ${eta.minutes}m ${eta.seconds}s`);
    console.log('─'.repeat(60) + '\n');
  }

  /**
   * Clear progress file
   */
  clearProgress(): void {
    try {
      const fs = require('fs');
      if (fs.existsSync(this.progressFile)) {
        fs.unlinkSync(this.progressFile);
        logger.info('Progress file cleared');
      }
    } catch (error) {
      logger.error('Failed to clear progress file', error);
    }
  }
}

// Export singleton instance
export const progressTracker = new ProgressTracker();
