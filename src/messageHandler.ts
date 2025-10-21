/**
 * Message handling and template processing
 */

import type { Customer, ProcessedCustomer } from './types';
import { config, getRandomTemplate } from '../config';
import { logger } from './logger';

/**
 * Render message template with customer data
 */
export function renderMessageTemplate(
  customer: ProcessedCustomer,
  template?: string
): string {
  const messageTemplate = template || getRandomTemplate();
  
  // Replace template variables
  let message = messageTemplate
    .replace(/{CustomerName}/g, customer.CustomerName)
    .replace(/{Code}/g, customer.Code)
    .replace(/{Phone}/g, customer.Phone)
    .replace(/{Mobile}/g, customer.Mobile);

  // Trim and validate length
  message = message.trim();
  
  if (message.length > config.message.maxLength) {
    logger.warning(
      `Message for ${customer.CustomerName} exceeds max length (${message.length}/${config.message.maxLength}), truncating`
    );
    message = message.substring(0, config.message.maxLength - 3) + '...';
  }

  return message;
}

/**
 * Preview message for a customer
 */
export function previewMessage(customer: ProcessedCustomer): void {
  const message = renderMessageTemplate(customer);
  
  console.log('\n' + '─'.repeat(60));
  console.log('MESSAGE PREVIEW');
  console.log('─'.repeat(60));
  console.log(`To: ${customer.CustomerName}`);
  console.log(`Phone: ${customer.formattedPhone}`);
  console.log(`Length: ${message.length} characters`);
  console.log('─'.repeat(60));
  console.log(message);
  console.log('─'.repeat(60) + '\n');
}

/**
 * Validate message content
 */
export function validateMessage(message: string): { valid: boolean; error?: string } {
  if (!message || message.trim() === '') {
    return { valid: false, error: 'Message is empty' };
  }

  if (message.length > config.message.maxLength) {
    return { 
      valid: false, 
      error: `Message exceeds maximum length (${message.length}/${config.message.maxLength})` 
    };
  }

  return { valid: true };
}

/**
 * Format phone number for WhatsApp (add @c.us suffix)
 */
export function formatWhatsAppNumber(phone: string): string {
  return `${phone}@c.us`;
}

/**
 * Estimate total execution time
 */
export function estimateExecutionTime(customerCount: number): {
  estimatedMinutes: number;
  estimatedSeconds: number;
  totalSeconds: number;
} {
  const avgDelay = (config.timing.delayBetweenMessages.min + config.timing.delayBetweenMessages.max) / 2;
  const batchCount = Math.ceil(customerCount / config.timing.batchSize);
  const batchDelayTotal = (batchCount - 1) * config.timing.delayBetweenBatches;
  
  const totalMs = (customerCount * avgDelay) + batchDelayTotal;
  const totalSeconds = Math.ceil(totalMs / 1000);
  const estimatedMinutes = Math.floor(totalSeconds / 60);
  const estimatedSeconds = totalSeconds % 60;

  return { estimatedMinutes, estimatedSeconds, totalSeconds };
}

/**
 * Display execution plan
 */
export function displayExecutionPlan(customerCount: number): void {
  const estimate = estimateExecutionTime(customerCount);
  const batchCount = Math.ceil(customerCount / config.timing.batchSize);

  console.log('\n' + '═'.repeat(60));
  console.log('EXECUTION PLAN');
  console.log('═'.repeat(60));
  console.log(`Total Customers:        ${customerCount}`);
  console.log(`Batch Size:             ${config.timing.batchSize} messages`);
  console.log(`Number of Batches:      ${batchCount}`);
  console.log(`Delay Between Messages: ${config.timing.delayBetweenMessages.min / 1000}-${config.timing.delayBetweenMessages.max / 1000}s`);
  console.log(`Delay Between Batches:  ${config.timing.delayBetweenBatches / 1000}s`);
  console.log(`Estimated Duration:     ${estimate.estimatedMinutes}m ${estimate.estimatedSeconds}s`);
  console.log(`Max Retries:            ${config.errorHandling.maxRetries}`);
  console.log('═'.repeat(60) + '\n');
}

/**
 * Create a delay promise
 */
export function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Format duration for display
 */
export function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);

  if (hours > 0) {
    return `${hours}h ${minutes % 60}m ${seconds % 60}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${seconds % 60}s`;
  } else {
    return `${seconds}s`;
  }
}
