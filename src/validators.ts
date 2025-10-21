/**
 * Data validation utilities
 */

import type { Customer, ValidationResult } from './types';
import { config } from '../config';

/**
 * Validate phone number format
 */
export function validatePhoneNumber(phone: string): ValidationResult {
  if (!phone || phone.trim() === '') {
    return { isValid: false, error: 'Phone number is empty' };
  }

  // Remove common invalid values
  const invalidPatterns = ['*****', '***', '65354', '10000000'];
  if (invalidPatterns.includes(phone.trim())) {
    return { isValid: false, error: 'Invalid phone number pattern' };
  }

  // Clean the phone number
  const cleaned = cleanPhoneNumber(phone);
  
  if (cleaned.length === 0) {
    return { isValid: false, error: 'Phone number contains no digits' };
  }

  // Format the phone number
  const formatted = formatPhoneNumber(cleaned);
  
  // Validate length
  const expectedLength = config.validation.phoneNumberLength;
  if (formatted.length !== expectedLength) {
    return { 
      isValid: false, 
      error: `Phone number length is ${formatted.length}, expected ${expectedLength}` 
    };
  }

  // Validate country code
  if (!formatted.startsWith(config.validation.countryCode)) {
    return { 
      isValid: false, 
      error: `Phone number must start with ${config.validation.countryCode}` 
    };
  }

  return { isValid: true, formattedValue: formatted };
}

/**
 * Clean phone number by removing non-numeric characters
 */
export function cleanPhoneNumber(phone: string): string {
  return phone.replace(/\D/g, '');
}

/**
 * Format phone number to WhatsApp format
 */
export function formatPhoneNumber(phone: string): string {
  const cleaned = cleanPhoneNumber(phone);
  
  // If number starts with 0, replace with country code
  if (cleaned.startsWith('0')) {
    return config.validation.countryCode + cleaned.substring(1);
  }
  
  // If number doesn't start with country code, add it
  if (!cleaned.startsWith(config.validation.countryCode)) {
    return config.validation.countryCode + cleaned;
  }
  
  return cleaned;
}

/**
 * Select the best phone number from customer data
 */
export function selectBestPhoneNumber(customer: Customer): string {
  const { Phone, Mobile } = customer;
  
  // Prefer mobile if configured
  if (config.validation.preferMobileOverPhone) {
    if (Mobile && Mobile.trim() !== '') {
      return Mobile;
    }
    return Phone;
  }
  
  // Otherwise prefer phone
  if (Phone && Phone.trim() !== '') {
    return Phone;
  }
  return Mobile;
}

/**
 * Validate customer data
 */
export function validateCustomerData(customer: Customer): ValidationResult {
  // Check if customer object exists
  if (!customer) {
    return { isValid: false, error: 'Customer data is null or undefined' };
  }

  // Check required fields
  if (!customer.CustomerName || customer.CustomerName.trim() === '') {
    return { isValid: false, error: 'Customer name is required' };
  }

  if (!customer.Code || customer.Code.trim() === '') {
    return { isValid: false, error: 'Customer code is required' };
  }

  // Check if at least one phone number exists
  const hasPhone = customer.Phone && customer.Phone.trim() !== '';
  const hasMobile = customer.Mobile && customer.Mobile.trim() !== '';
  
  if (!hasPhone && !hasMobile) {
    return { isValid: false, error: 'No phone number provided' };
  }

  return { isValid: true };
}

/**
 * Validate CSV structure
 */
export function validateCSVStructure(headers: string[]): ValidationResult {
  const requiredHeaders = ['Code', 'CustomerName', 'Phone', 'Mobile'];
  const missingHeaders = requiredHeaders.filter(h => !headers.includes(h));
  
  if (missingHeaders.length > 0) {
    return { 
      isValid: false, 
      error: `Missing required columns: ${missingHeaders.join(', ')}` 
    };
  }

  return { isValid: true };
}

/**
 * Sanitize customer name for message
 */
export function sanitizeCustomerName(name: string): string {
  if (!name) return '';
  
  // Remove extra whitespace
  let sanitized = name.trim().replace(/\s+/g, ' ');
  
  // Remove common prefixes/suffixes that might cause issues
  sanitized = sanitized.replace(/\s*\(.*?\)\s*/g, ' ').trim();
  
  return sanitized;
}

/**
 * Check if customer should be skipped
 */
export function shouldSkipCustomer(customer: Customer): { skip: boolean; reason?: string } {
  // Check for special orders or invalid entries
  if (customer.CustomerName.toUpperCase().includes('SPECIAL ORDER')) {
    return { skip: true, reason: 'Special order entry' };
  }

  // Check for test entries
  if (customer.Code === '0' || customer.Code === '00') {
    return { skip: true, reason: 'Test entry' };
  }

  return { skip: false };
}
