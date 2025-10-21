/**
 * Type definitions for the bulk messaging system
 */

export interface Customer {
  Code: string;
  CustomerName: string;
  Phone: string;
  Mobile: string;
}

export interface ProcessedCustomer extends Customer {
  selectedPhone: string;
  formattedPhone: string;
  isValid: boolean;
  validationError?: string;
}

export interface MessageResult {
  customer: ProcessedCustomer;
  success: boolean;
  timestamp: Date;
  error?: string;
  retryCount: number;
}

export interface Config {
  // Session settings
  session: {
    name: string;
    headless: boolean;
    autoClose: number;
    disableSpins: boolean;
    disableWelcome: boolean;
  };

  // Timing settings
  timing: {
    delayBetweenMessages: {
      min: number;
      max: number;
    };
    batchSize: number;
    delayBetweenBatches: number;
    initialWarmupDelay: number;
    retryDelay: number;
  };

  // Message settings
  message: {
    templateFile: string;
    defaultTemplate: string;
    maxLength: number;
  };

  // Error handling
  errorHandling: {
    maxRetries: number;
    continueOnError: boolean;
    saveFailedCustomers: boolean;
  };

  // Logging
  logging: {
    enabled: boolean;
    logLevel: 'info' | 'warning' | 'error' | 'debug';
    logToFile: boolean;
    logToConsole: boolean;
    logDirectory: string;
  };

  // Data validation
  validation: {
    countryCode: string;
    phoneNumberLength: number;
    skipInvalidNumbers: boolean;
    preferMobileOverPhone: boolean;
  };

  // Rate limiting
  rateLimiting: {
    enabled: boolean;
    maxMessagesPerDay: number;
    maxMessagesPerHour: number;
  };
}

export interface ProgressState {
  totalCustomers: number;
  processedCustomers: number;
  successfulSends: number;
  failedSends: number;
  skippedCustomers: number;
  startTime: Date;
  lastProcessedIndex: number;
  currentBatch: number;
}

export interface ExecutionReport {
  startTime: Date;
  endTime: Date;
  duration: number;
  totalCustomers: number;
  successfulSends: number;
  failedSends: number;
  skippedCustomers: number;
  successRate: number;
  averageDelay: number;
  errors: Array<{
    customer: string;
    error: string;
    timestamp: Date;
  }>;
}

export interface LogEntry {
  timestamp: Date;
  level: 'info' | 'warning' | 'error' | 'success' | 'debug';
  message: string;
  data?: any;
}

export interface ValidationResult {
  isValid: boolean;
  error?: string;
  formattedValue?: string;
}
