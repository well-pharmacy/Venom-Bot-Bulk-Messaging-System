/**
 * Configuration file for Venom-Bot bulk messaging system
 * Adjust these settings according to your needs
 */

import type { Config } from './src/types';

export const config: Config = {
  // Session settings
  session: {
    name: 'bulk-messaging-session',
    headless: false, // Set to true for production, false to see browser
    autoClose: 60000, // Close browser after 60 seconds of inactivity
    disableSpins: true,
    disableWelcome: true,
  },

  // Timing settings - CRITICAL for avoiding bans
  timing: {
    // Random delay between messages (in milliseconds)
    delayBetweenMessages: {
      min: 5000,  // 5 seconds minimum
      max: 12000, // 12 seconds maximum
    },

    // Process messages in batches to simulate human behavior
    batchSize: 20, // Send 20 messages, then take a break
    delayBetweenBatches: 120000, // 2 minutes break between batches

    // Initial warmup - slower sending for first few messages
    initialWarmupDelay: 15000, // 15 seconds for first 5 messages

    // Delay before retrying failed messages
    retryDelay: 30000, // 30 seconds
  },

  // Message settings
  message: {
    templateFile: './templates/messages.json',
    defaultTemplate: 'مرحباً {CustomerName}،\n\nنود أن نشكرك على كونك عميلاً مميزاً لدينا.\n\nرقم العميل: {Code}\n\nنتطلع لخدمتك دائماً.',
    maxLength: 1000, // Maximum message length
  },

  // Error handling
  errorHandling: {
    maxRetries: 3, // Retry failed messages up to 3 times
    continueOnError: true, // Continue sending even if some messages fail
    saveFailedCustomers: true, // Save failed customers to CSV for retry
  },

  // Logging
  logging: {
    enabled: true,
    logLevel: 'info', // 'debug' | 'info' | 'warning' | 'error'
    logToFile: true,
    logToConsole: true,
    logDirectory: './logs',
  },

  // Data validation
  validation: {
    countryCode: '20', // Egypt country code
    phoneNumberLength: 12, // 20 + 10 digits (e.g., 201234567890)
    skipInvalidNumbers: true, // Skip customers with invalid numbers
    preferMobileOverPhone: true, // Use Mobile column if both exist
  },

  // Rate limiting - IMPORTANT for account safety
  rateLimiting: {
    enabled: true,
    maxMessagesPerDay: 500, // Don't exceed 500 messages per day
    maxMessagesPerHour: 100, // Don't exceed 100 messages per hour
  },
};

/**
 * Message templates with variables
 * Variables: {CustomerName}, {Code}, {Phone}, {Mobile}
 */
export const messageTemplates = [
  'مرحباً {CustomerName}،\n\nشكراً لك على ثقتك بنا.\nرقم العميل: {Code}',

  'عزيزي {CustomerName}،\n\nنحن سعداء بخدمتك.\nكود العميل: {Code}',

  'أهلاً {CustomerName}،\n\nنتمنى أن تكون بخير.\nرقمك لدينا: {Code}',
];

/**
 * Get a random delay within the configured range
 */
export function getRandomDelay(isWarmup: boolean = false): number {
  if (isWarmup) {
    return config.timing.initialWarmupDelay;
  }

  const { min, max } = config.timing.delayBetweenMessages;
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

/**
 * Check if we should take a batch break
 */
export function shouldTakeBatchBreak(messageCount: number): boolean {
  return messageCount > 0 && messageCount % config.timing.batchSize === 0;
}

/**
 * Get a random message template
 */
export function getRandomTemplate(): string {
  const templates = messageTemplates.length > 0
    ? messageTemplates
    : [config.message.defaultTemplate];

  return templates[Math.floor(Math.random() * templates.length)]!;
}
