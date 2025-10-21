/**
 * Venom-Bot Bulk Messaging System
 * Main entry point for sending WhatsApp messages to customers from CSV
 */

import { create, Whatsapp } from 'venom-bot';
import { parseCSV, saveFailedCustomers, getCSVStats } from './src/csvParser';
import {
    validatePhoneNumber,
    validateCustomerData,
    selectBestPhoneNumber,
    shouldSkipCustomer,
    sanitizeCustomerName
} from './src/validators';
import { logger } from './src/logger';
import { progressTracker } from './src/progressTracker';
import {
    renderMessageTemplate,
    formatWhatsAppNumber,
    displayExecutionPlan,
    delay,
    previewMessage
} from './src/messageHandler';
import { config, getRandomDelay, shouldTakeBatchBreak } from './config';
import type { Customer, ProcessedCustomer, MessageResult } from './src/types';

// Global state
let client: Whatsapp;
let isShuttingDown = false;

/**
 * Main execution function
 */
async function main() {
    try {
        logger.info('Starting Venom-Bot Bulk Messaging System');
        logger.info('Configuration loaded', {
            batchSize: config.timing.batchSize,
            delayRange: `${config.timing.delayBetweenMessages.min}-${config.timing.delayBetweenMessages.max}ms`
        });

        // Load CSV data
        const csvPath = './customers.csv';
        const customers = parseCSV(csvPath);

        if (customers.length === 0) {
            logger.error('No customers found in CSV file');
            return;
        }

        // Display CSV statistics
        const stats = getCSVStats(customers);
        logger.info('CSV Statistics', stats);

        // Process and validate customers
        const processedCustomers = processCustomers(customers);

        if (processedCustomers.length === 0) {
            logger.error('No valid customers to process after validation');
            return;
        }

        logger.info(`Valid customers ready for processing: ${processedCustomers.length}`);

        // Display execution plan
        displayExecutionPlan(processedCustomers.length);

        // Preview first message
        if (processedCustomers.length > 0) {
            const firstCustomer = processedCustomers[0];
            if (firstCustomer) {
                logger.info('Previewing first message:');
                previewMessage(firstCustomer);
            }
        }

        // Ask for confirmation (in production, you might want to add a prompt here)
        logger.info('Starting message sending in 5 seconds...');
        await delay(5000);

        // Initialize Venom-Bot
        logger.info('Initializing Venom-Bot...');
        client = await initializeVenomBot();
        logger.success('Venom-Bot initialized successfully');

        // Initialize progress tracker
        progressTracker.initialize(processedCustomers.length);

        // Send messages
        await sendMessagesToCustomers(client, processedCustomers);

        // Generate final report
        const report = progressTracker.generateReport();
        logger.generateReport(report);

        // Save failed customers if any
        if (config.errorHandling.saveFailedCustomers) {
            const failedCustomers = progressTracker.getFailedCustomers();
            if (failedCustomers.length > 0) {
                saveFailedCustomers(failedCustomers, './data/failed-customers.csv');
            }
        }

        // Cleanup
        await cleanup();

        logger.success('Bulk messaging completed successfully');
    } catch (error) {
        logger.error('Fatal error in main execution', error);
        await cleanup();
        process.exit(1);
    }
}

/**
 * Initialize Venom-Bot client
 */
async function initializeVenomBot(): Promise<Whatsapp> {
    return await create(
        config.session.name,
        (base64Qr) => {
            logger.info('QR Code received, scan with WhatsApp');
            console.log('\n');
            console.log(base64Qr);
            console.log('\n');
        },
        (statusSession) => {
            logger.info(`Session status: ${statusSession}`);
        },
        {
            headless: config.session.headless ? 'new' : false,
            devtools: false,
            debug: false,
            logQR: true,
            browserArgs: [
                '--no-sandbox',
                '--disable-setuid-sandbox',
                '--disable-dev-shm-usage',
                '--disable-accelerated-2d-canvas',
                '--no-first-run',
                '--no-zygote',
                '--disable-gpu'
            ],
            autoClose: config.session.autoClose,
            disableSpins: config.session.disableSpins,
            disableWelcome: config.session.disableWelcome,
        }
    );
}

/**
 * Process and validate all customers
 */
function processCustomers(customers: Customer[]): ProcessedCustomer[] {
    const processed: ProcessedCustomer[] = [];

    for (const customer of customers) {
        // Check if should skip
        const skipCheck = shouldSkipCustomer(customer);
        if (skipCheck.skip) {
            logger.warning(`Skipping customer ${customer.CustomerName}: ${skipCheck.reason}`);
            continue;
        }

        // Validate customer data
        const dataValidation = validateCustomerData(customer);
        if (!dataValidation.isValid) {
            logger.warning(`Invalid customer data for ${customer.CustomerName}: ${dataValidation.error}`);
            continue;
        }

        // Select best phone number
        const selectedPhone = selectBestPhoneNumber(customer);

        // Validate phone number
        const phoneValidation = validatePhoneNumber(selectedPhone);

        const processedCustomer: ProcessedCustomer = {
            ...customer,
            CustomerName: sanitizeCustomerName(customer.CustomerName),
            selectedPhone,
            formattedPhone: phoneValidation.formattedValue || '',
            isValid: phoneValidation.isValid,
            validationError: phoneValidation.error,
        };

        if (!processedCustomer.isValid) {
            if (config.validation.skipInvalidNumbers) {
                logger.warning(
                    `Skipping ${customer.CustomerName} - Invalid phone: ${phoneValidation.error}`
                );
                continue;
            } else {
                logger.warning(
                    `Customer ${customer.CustomerName} has invalid phone but will be attempted: ${phoneValidation.error}`
                );
            }
        }

        processed.push(processedCustomer);
    }

    return processed;
}

/**
 * Send messages to all customers
 */
async function sendMessagesToCustomers(
    client: Whatsapp,
    customers: ProcessedCustomer[]
): Promise<void> {
    logger.info(`Starting to send messages to ${customers.length} customers`);

    for (let i = 0; i < customers.length; i++) {
        if (isShuttingDown) {
            logger.warning('Shutdown requested, stopping message sending');
            break;
        }

        const customer = customers[i];
        if (!customer) continue;

        const isWarmup = i < 5; // First 5 messages use warmup delay

        // Display progress
        logger.displayProgress(i + 1, customers.length, customer.CustomerName);

        // Send message with retry logic
        const result = await sendMessageWithRetry(client, customer, isWarmup);

        // Calculate delay for this message
        const delayMs = getRandomDelay(isWarmup);

        // Record result
        progressTracker.recordResult(result, delayMs);
        logger.logMessageResult(result);

        // Check if we should take a batch break
        if (shouldTakeBatchBreak(i + 1)) {
            progressTracker.incrementBatch();
            logger.clearProgress();
            logger.info(
                `Batch ${progressTracker.getState().currentBatch} completed. Taking ${config.timing.delayBetweenBatches / 1000}s break...`
            );
            progressTracker.displayStats();
            await delay(config.timing.delayBetweenBatches);
            logger.info('Resuming message sending...');
        } else {
            // Regular delay between messages
            await delay(delayMs);
        }
    }

    logger.clearProgress();
    logger.success('All messages processed');
}

/**
 * Send message with retry logic
 */
async function sendMessageWithRetry(
    client: Whatsapp,
    customer: ProcessedCustomer,
    isWarmup: boolean
): Promise<MessageResult> {
    let lastError: string = '';

    for (let attempt = 0; attempt <= config.errorHandling.maxRetries; attempt++) {
        try {
            // Render message
            const message = renderMessageTemplate(customer);
            const whatsappNumber = formatWhatsAppNumber(customer.formattedPhone);

            // Check if number exists on WhatsApp
            const numberExists = await client.checkNumberStatus(whatsappNumber);

            if (!numberExists.numberExists) {
                return {
                    customer,
                    success: false,
                    timestamp: new Date(),
                    error: 'Phone number not registered on WhatsApp',
                    retryCount: attempt,
                };
            }

            // Send message
            await client.sendText(whatsappNumber, message);

            return {
                customer,
                success: true,
                timestamp: new Date(),
                retryCount: attempt,
            };
        } catch (error: any) {
            lastError = error?.message || String(error);

            if (attempt < config.errorHandling.maxRetries) {
                logger.warning(
                    `Attempt ${attempt + 1} failed for ${customer.CustomerName}, retrying...`
                );
                await delay(config.timing.retryDelay);
            }
        }
    }

    return {
        customer,
        success: false,
        timestamp: new Date(),
        error: lastError,
        retryCount: config.errorHandling.maxRetries,
    };
}

/**
 * Cleanup and close connections
 */
async function cleanup(): Promise<void> {
    try {
        if (client) {
            logger.info('Closing Venom-Bot session...');
            await client.close();
            logger.info('Session closed');
        }
    } catch (error) {
        logger.error('Error during cleanup', error);
    }
}

/**
 * Handle graceful shutdown
 */
function setupGracefulShutdown(): void {
    const shutdownHandler = async (signal: string) => {
        if (isShuttingDown) return;

        isShuttingDown = true;
        logger.warning(`Received ${signal}, shutting down gracefully...`);

        // Save current progress
        progressTracker.saveProgress();

        await cleanup();
        process.exit(0);
    };

    process.on('SIGINT', () => shutdownHandler('SIGINT'));
    process.on('SIGTERM', () => shutdownHandler('SIGTERM'));
}

// Setup graceful shutdown
setupGracefulShutdown();

// Run main function
main().catch((error) => {
    logger.error('Unhandled error', error);
    process.exit(1);
});