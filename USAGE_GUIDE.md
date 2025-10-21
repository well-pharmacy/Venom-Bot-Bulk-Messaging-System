# Venom-Bot Bulk Messaging - Usage Guide

## Quick Start

### 1. Installation

```bash
# Install dependencies
bun install
```

### 2. Prepare Your CSV File

Ensure your `customers.csv` file has the following structure:

```csv
Code,CustomerName,Phone,Mobile
1,أحمد محمد,201234567890,201234567890
2,فاطمة علي,201098765432,
```

**Required Columns:**
- `Code`: Customer ID/Code
- `CustomerName`: Customer name (supports Arabic)
- `Phone`: Primary phone number
- `Mobile`: Mobile number (optional if Phone is provided)

### 3. Configure Settings

Edit `config.ts` to adjust:

```typescript
timing: {
  delayBetweenMessages: {
    min: 5000,  // 5 seconds
    max: 12000, // 12 seconds
  },
  batchSize: 20, // Messages per batch
  delayBetweenBatches: 120000, // 2 minutes
}
```

### 4. Customize Message Templates

Edit message templates in `config.ts`:

```typescript
export const messageTemplates = [
  'مرحباً {CustomerName}، رقم حسابك: {Code}',
  'عزيزي {CustomerName}، شكراً لثقتك بنا. الكود: {Code}',
];
```

**Available Variables:**
- `{CustomerName}` - Customer name
- `{Code}` - Customer code
- `{Phone}` - Phone number
- `{Mobile}` - Mobile number

### 5. Run the Script

```bash
# Start the messaging system
bun run index.ts
```

## Step-by-Step Execution

### Phase 1: Initialization
1. Script loads and validates configuration
2. Reads and parses CSV file
3. Validates customer data and phone numbers
4. Displays execution plan with estimates

### Phase 2: WhatsApp Connection
1. Venom-Bot initializes
2. QR code displays in console
3. Scan QR code with WhatsApp mobile app
4. Wait for connection confirmation

### Phase 3: Message Sending
1. Preview of first message shown
2. 5-second countdown before starting
3. Messages sent with random delays
4. Progress displayed in real-time
5. Batch breaks taken automatically
6. Errors logged and retried

### Phase 4: Completion
1. Final statistics displayed
2. Execution report generated
3. Failed customers saved to CSV
4. Session closed gracefully

## Configuration Options

### Timing Settings

```typescript
timing: {
  delayBetweenMessages: {
    min: 5000,   // Minimum delay (ms)
    max: 12000,  // Maximum delay (ms)
  },
  batchSize: 20,              // Messages before break
  delayBetweenBatches: 120000, // Break duration (ms)
  initialWarmupDelay: 15000,   // Delay for first 5 messages
  retryDelay: 30000,           // Delay before retry
}
```

**Recommendations:**
- **Small batches (<50)**: 5-10s delay, batch size 10-15
- **Medium batches (50-200)**: 8-15s delay, batch size 20-30
- **Large batches (>200)**: 10-20s delay, batch size 20-25

### Session Settings

```typescript
session: {
  name: 'bulk-messaging-session', // Session name
  headless: false,                // Show browser (false) or hide (true)
  autoClose: 60000,               // Auto-close after inactivity
  disableSpins: true,             // Disable loading spinners
  disableWelcome: true,           // Disable welcome message
}
```

### Error Handling

```typescript
errorHandling: {
  maxRetries: 3,              // Retry attempts per message
  continueOnError: true,      // Continue if message fails
  saveFailedCustomers: true,  // Save failed to CSV
}
```

### Validation Settings

```typescript
validation: {
  countryCode: '20',              // Egypt (change for other countries)
  phoneNumberLength: 12,          // Expected length (20 + 10 digits)
  skipInvalidNumbers: true,       // Skip invalid numbers
  preferMobileOverPhone: true,    // Use Mobile column first
}
```

### Rate Limiting

```typescript
rateLimiting: {
  enabled: true,
  maxMessagesPerDay: 500,   // Daily limit
  maxMessagesPerHour: 100,  // Hourly limit
}
```

## Phone Number Formats

The system automatically formats phone numbers. Supported formats:

```
201234567890  ✓ (Preferred format)
01234567890   ✓ (Converted to 201234567890)
1234567890    ✓ (Country code added)
+201234567890 ✓ (Plus sign removed)
```

**Invalid formats:**
```
*****         ✗
***           ✗
65354         ✗
10000000      ✗
```

## Message Templates

### Using Variables

```typescript
const template = 'مرحباً {CustomerName}، رقم حسابك: {Code}';
```

### Multiple Templates

The system randomly selects from available templates:

```typescript
export const messageTemplates = [
  'Template 1 with {CustomerName}',
  'Template 2 with {Code}',
  'Template 3 with {CustomerName} and {Code}',
];
```

### Template Best Practices

1. **Keep it concise** - Under 1000 characters
2. **Personalize** - Use customer name
3. **Be clear** - State purpose clearly
4. **Add value** - Provide useful information
5. **Include opt-out** - Respect customer preferences

## Monitoring Progress

### Real-Time Display

```
[████████████████░░░░░░░░░░░░░░] 55.2% (138/250) - Processing: أحمد محمد
```

### Statistics Display

Every batch break shows:
- Processed count
- Successful sends
- Failed sends
- Skipped customers
- Success rate
- Current batch
- Estimated time remaining

### Log Files

Located in `./logs/`:
- `app-YYYY-MM-DD.log` - All operations
- `errors-YYYY-MM-DD.log` - Errors only
- `success-YYYY-MM-DD.log` - Successful sends

## Error Handling

### Common Errors

#### "Phone number not registered on WhatsApp"
- **Cause**: Number doesn't have WhatsApp
- **Action**: Skipped automatically, logged

#### "Connection lost"
- **Cause**: Internet or WhatsApp connection issue
- **Action**: Automatic retry, then fail

#### "Invalid phone number"
- **Cause**: Wrong format or length
- **Action**: Skipped if `skipInvalidNumbers: true`

#### "Rate limit exceeded"
- **Cause**: Too many messages too fast
- **Action**: Increase delays, reduce batch size

### Retry Logic

1. **First attempt** - Send message
2. **If fails** - Wait 30 seconds
3. **Second attempt** - Retry
4. **If fails** - Wait 30 seconds
5. **Third attempt** - Final retry
6. **If fails** - Mark as failed, continue

## Failed Customers

Failed customers are saved to `./data/failed-customers.csv`:

```csv
Code,CustomerName,Phone,Mobile
5,محمد أحمد,201234567890,201234567890
```

### Retry Failed Customers

1. Copy `failed-customers.csv` to `customers.csv`
2. Run script again
3. Only failed customers will be processed

## Safety Features

### Anti-Ban Measures

1. **Random delays** - Mimics human behavior
2. **Batch processing** - Regular breaks
3. **Warmup period** - Slower start
4. **Daily limits** - Prevents overuse
5. **Number validation** - Checks before sending

### Graceful Shutdown

Press `Ctrl+C` to stop:
1. Current message completes
2. Progress saved
3. Session closed properly
4. Can resume later

## Troubleshooting

### QR Code Not Showing

**Solution**: Set `headless: false` in config

### Messages Not Sending

**Check**:
1. WhatsApp is connected
2. Phone numbers are valid
3. Internet connection is stable
4. No WhatsApp restrictions

### High Failure Rate

**Actions**:
1. Increase delays between messages
2. Reduce batch size
3. Check phone number format
4. Verify CSV data quality

### Script Crashes

**Check**:
1. Log files for errors
2. CSV file encoding (UTF-8)
3. Available memory
4. Node.js/Bun version

## Best Practices

### Before Running

- [ ] Test with 5-10 customers first
- [ ] Verify phone number formats
- [ ] Review message templates
- [ ] Check CSV encoding (UTF-8)
- [ ] Backup customer data
- [ ] Set appropriate delays

### During Execution

- [ ] Monitor first 10-20 messages
- [ ] Check success rate
- [ ] Watch for errors
- [ ] Don't close terminal
- [ ] Keep internet stable

### After Completion

- [ ] Review execution report
- [ ] Check failed customers
- [ ] Analyze error logs
- [ ] Update customer records
- [ ] Clean old log files

## Performance Tips

### For Large Batches (>500)

1. **Split into multiple days** - 500 per day max
2. **Increase delays** - 15-20 seconds
3. **Longer breaks** - 5-10 minutes
4. **Monitor closely** - Watch for issues

### For Better Delivery

1. **Verify numbers** - Clean data first
2. **Personalize messages** - Use names
3. **Vary templates** - Multiple versions
4. **Optimal timing** - Send during business hours
5. **Test first** - Small batch validation

## Advanced Usage

### Custom Validation

Edit `src/validators.ts` to add custom rules:

```typescript
export function validateCustomerData(customer: Customer): ValidationResult {
  // Add custom validation logic
  if (customer.CustomerName.length < 3) {
    return { isValid: false, error: 'Name too short' };
  }
  return { isValid: true };
}
```

### Custom Message Logic

Edit `src/messageHandler.ts`:

```typescript
export function renderMessageTemplate(customer: ProcessedCustomer): string {
  // Add custom logic
  const isVIP = customer.CustomerName.includes('VIP');
  const template = isVIP ? vipTemplate : regularTemplate;
  return template.replace(/{CustomerName}/g, customer.CustomerName);
}
```

### Resume Functionality

Progress is auto-saved every 10 messages to `./data/progress.json`. To resume:

1. Don't delete progress file
2. Run script again
3. It will continue from last successful send

## Support & Maintenance

### Regular Maintenance

- **Daily**: Review error logs
- **Weekly**: Clean old logs, update failed customers
- **Monthly**: Review success rates, optimize settings

### Backup Strategy

Backup these files regularly:
- `customers.csv` - Customer data
- `config.ts` - Configuration
- `templates/messages.json` - Message templates
- `logs/` - Log files (for audit)

### Updates

Keep dependencies updated:

```bash
bun update
```

## Legal & Compliance

### Important Notes

1. **Consent**: Only message customers who opted in
2. **Opt-out**: Provide unsubscribe mechanism
3. **Privacy**: Protect customer data
4. **Frequency**: Don't spam customers
5. **Content**: Keep messages relevant
6. **Compliance**: Follow local regulations

### WhatsApp Terms

- Don't exceed reasonable message limits
- Don't send spam or unsolicited messages
- Don't use for illegal purposes
- Respect user privacy
- Follow WhatsApp Business Policy

## Getting Help

### Check Logs

```bash
# View latest log
cat logs/app-YYYY-MM-DD.log

# View errors only
cat logs/errors-YYYY-MM-DD.log
```

### Debug Mode

Set `logLevel: 'debug'` in config for detailed logs.

### Common Solutions

1. **Restart script** - Fixes most issues
2. **Check internet** - Stable connection needed
3. **Verify CSV** - Correct format and encoding
4. **Update dependencies** - `bun update`
5. **Clear session** - Delete session files

---

**Version**: 1.0.0  
**Last Updated**: 2025-10-21  
**Author**: Venom-Bot Bulk Messaging System
