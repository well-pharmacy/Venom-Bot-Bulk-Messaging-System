/**
 * CSV parsing utilities
 */

import { readFileSync } from 'fs';
import type { Customer } from './types';
import { logger } from './logger';
import { validateCSVStructure } from './validators';

/**
 * Parse CSV file and return customer data
 */
export function parseCSV(filePath: string): Customer[] {
  try {
    logger.info(`Reading CSV file: ${filePath}`);

    // Read file with UTF-8 encoding to handle Arabic characters
    const fileContent = readFileSync(filePath, 'utf-8');

    // Split into lines
    const lines = fileContent.split('\n').filter(line => line.trim() !== '');

    if (lines.length === 0) {
      throw new Error('CSV file is empty');
    }

    // Parse header
    const headerLine = lines[0];
    if (!headerLine) {
      throw new Error('CSV header is missing');
    }
    const headers = parseCSVLine(headerLine).filter((h): h is string => h !== undefined);
    logger.debug('CSV Headers', headers);

    // Validate structure
    const validation = validateCSVStructure(headers);
    if (!validation.isValid) {
      throw new Error(validation.error);
    }

    // Parse data rows
    const customers: Customer[] = [];
    const duplicates = new Set<string>();
    const seenCodes = new Set<string>();

    for (let i = 1; i < lines.length; i++) {
      try {
        const values = parseCSVLine(lines[i]!);

        if (values.length < headers.length) {
          logger.warning(`Line ${i + 1}: Incomplete data, skipping`);
          continue;
        }

        // Create customer object
        const customer: any = {};
        headers.forEach((header, index) => {
          const value = values[index];
          customer[header] = value !== undefined ? value : '';
        });

        // Check for duplicates
        if (seenCodes.has(customer.Code)) {
          duplicates.add(customer.Code);
          logger.warning(`Duplicate customer code found: ${customer.Code}`);
          continue;
        }

        seenCodes.add(customer.Code);
        customers.push(customer as Customer);
      } catch (error) {
        logger.warning(`Line ${i + 1}: Parse error - ${error}`);
      }
    }

    logger.info(`Successfully parsed ${customers.length} customers from CSV`);

    if (duplicates.size > 0) {
      logger.warning(`Found ${duplicates.size} duplicate entries`);
    }

    return customers;
  } catch (error) {
    logger.error('Failed to parse CSV file', error);
    throw error;
  }
}

/**
 * Parse a single CSV line, handling quoted values
 */
function parseCSVLine(line: string): string[] {
  const values: string[] = [];
  let currentValue = '';
  let insideQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    const nextChar = line[i + 1];

    if (char === '"') {
      if (insideQuotes && nextChar === '"') {
        // Escaped quote
        currentValue += '"';
        i++; // Skip next quote
      } else {
        // Toggle quote state
        insideQuotes = !insideQuotes;
      }
    } else if (char === ',' && !insideQuotes) {
      // End of value
      values.push(currentValue.trim());
      currentValue = '';
    } else {
      currentValue += char;
    }
  }

  // Add last value
  values.push(currentValue.trim());

  return values;
}

/**
 * Save failed customers to CSV for retry
 */
export function saveFailedCustomers(customers: Customer[], filePath: string): void {
  try {
    const headers = 'Code,CustomerName,Phone,Mobile\n';
    const rows = customers.map(c =>
      `${c.Code},"${c.CustomerName}",${c.Phone},${c.Mobile}`
    ).join('\n');

    const content = headers + rows;

    const fs = require('fs');
    fs.writeFileSync(filePath, content, 'utf-8');

    logger.info(`Saved ${customers.length} failed customers to ${filePath}`);
  } catch (error) {
    logger.error('Failed to save failed customers', error);
  }
}

/**
 * Get CSV file statistics
 */
export function getCSVStats(customers: Customer[]): {
  total: number;
  withPhone: number;
  withMobile: number;
  withBoth: number;
  withNeither: number;
} {
  const stats = {
    total: customers.length,
    withPhone: 0,
    withMobile: 0,
    withBoth: 0,
    withNeither: 0,
  };

  customers.forEach(customer => {
    const hasPhone = customer.Phone && customer.Phone.trim() !== '';
    const hasMobile = customer.Mobile && customer.Mobile.trim() !== '';

    if (hasPhone && hasMobile) {
      stats.withBoth++;
    } else if (hasPhone) {
      stats.withPhone++;
    } else if (hasMobile) {
      stats.withMobile++;
    } else {
      stats.withNeither++;
    }
  });

  return stats;
}
