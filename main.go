package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/manifoldco/promptui"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

// Color codes for console output
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorGray    = "\033[90m"

	// Bright colors
	colorBrightRed    = "\033[91m"
	colorBrightGreen  = "\033[92m"
	colorBrightYellow = "\033[93m"
	colorBrightCyan   = "\033[96m"

	// Background colors
	bgRed    = "\033[41m"
	bgGreen  = "\033[42m"
	bgYellow = "\033[43m"

	// Text styles
	bold      = "\033[1m"
	dim       = "\033[2m"
	underline = "\033[4m"
	blink     = "\033[5m"
)

// Logger handles logging to console and files
type logger struct {
	logFile     *os.File
	errorFile   *os.File
	successFile *os.File
}

// NewLogger creates a new logger instance
func NewLogger() *logger {
	// Create logs directory
	os.MkdirAll("logs", 0755)

	timestamp := time.Now().Format("2006-01-02")

	// Open log files
	logFile, _ := os.OpenFile(
		filepath.Join("logs", fmt.Sprintf("app-%s.log", timestamp)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)

	errorFile, _ := os.OpenFile(
		filepath.Join("logs", fmt.Sprintf("errors-%s.log", timestamp)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)

	successFile, _ := os.OpenFile(
		filepath.Join("logs", fmt.Sprintf("success-%s.log", timestamp)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)

	return &logger{
		logFile:     logFile,
		errorFile:   errorFile,
		successFile: successFile,
	}
}

// log writes a log entry
func (l *logger) log(level, color, message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Console output with color
	fmt.Printf("%s[%s] %-7s%s %s\n", color, timestamp, level, colorReset, message)

	// File output
	if l.logFile != nil {
		logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)
		l.logFile.WriteString(logLine)
	}
}

// Info logs an info message
func (l *logger) Info(message string) {
	l.log("INFO", colorCyan, message)
}

// Success logs a success message
func (l *logger) Success(message string) {
	l.log("SUCCESS", colorGreen, message)

	// Also write to success file
	if l.successFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		l.successFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
	}
}

// Warning logs a warning message
func (l *logger) Warning(message string) {
	l.log("WARNING", colorYellow, message)
}

// Error logs an error message
func (l *logger) Error(message string, err error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	l.log("ERROR", colorRed, errorMsg)

	// Also write to error file
	if l.errorFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		l.errorFile.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, errorMsg))
	}
}

// Debug logs a debug message
func (l *logger) Debug(message string) {
	l.log("DEBUG", colorGray, message)
}

// Close closes all log files
func (l *logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
	if l.errorFile != nil {
		l.errorFile.Close()
	}
	if l.successFile != nil {
		l.successFile.Close()
	}
}

// Customer represents a customer record from CSV
type Customer struct {
	Code         string
	CustomerName string
	Phone        string
	Mobile       string
	HasWhatsApp  string // "yes", "no", or "" (unchecked)
}

// ProcessedCustomer represents a validated customer
type ProcessedCustomer struct {
	Customer
	SelectedPhone   string
	FormattedPhone  string
	IsValid         bool
	ValidationError string
}

// MessageResult represents the result of sending a message
type MessageResult struct {
	Customer   ProcessedCustomer
	Success    bool
	Timestamp  time.Time
	Error      string
	RetryCount int
}

// Config holds application configuration
type Config struct {
	DelayMin        int
	DelayMax        int
	BatchSize       int
	BatchDelay      int
	WarmupDelay     int
	RetryDelay      int
	MaxRetries      int
	CountryCode     string
	PhoneLength     int
	SkipInvalid     bool
	PreferMobile    bool
	ContinueOnError bool
	SaveFailed      bool
	SkipDuplicates  bool // Skip duplicate phone numbers
	PreCheckNumbers bool // Pre-check all numbers before sending
	CheckDelay      int  // Delay between checks (milliseconds)

	// Anti-blocking features
	HourlyLimit       int     // Max messages per hour
	DailyLimit        int     // Max messages per day
	BusinessHoursOnly bool    // Only send during business hours (9 AM - 9 PM)
	SimulateTyping    bool    // Simulate typing before sending
	AddJitter         bool    // Add random micro-delays
	LongPauseChance   float32 // Chance of taking a long pause (0.0-1.0)
}

// ProgressTracker tracks messaging progress
type ProgressTracker struct {
	Total      int
	Processed  int
	Successful int
	Failed     int
	Skipped    int
	Duplicates int // Count of duplicate phone numbers
	StartTime  time.Time
	Delays     []int

	// Rate limiting
	HourlySent    int
	DailySent     int
	LastHourReset time.Time
	LastDayReset  time.Time
}

var (
	config = Config{
		DelayMin:        5000,
		DelayMax:        12000,
		BatchSize:       20,
		BatchDelay:      120000,
		WarmupDelay:     15000,
		RetryDelay:      30000,
		MaxRetries:      3,
		CountryCode:     "20",
		PhoneLength:     12,
		SkipInvalid:     true,
		PreferMobile:    true,
		ContinueOnError: true,
		SaveFailed:      true,
		SkipDuplicates:  true,  // Skip duplicate phone numbers by default
		PreCheckNumbers: false, // Don't pre-check by default (to avoid rate limiting)
		CheckDelay:      2000,  // 2 seconds between checks

		// Anti-blocking defaults
		HourlyLimit:       100,  // Max 100 messages per hour
		DailyLimit:        500,  // Max 500 messages per day
		BusinessHoursOnly: true, // Only send during business hours
		SimulateTyping:    true, // Simulate typing
		AddJitter:         true, // Add random micro-delays
		LongPauseChance:   0.05, // 5% chance of long pause
	}

	progress = &ProgressTracker{
		StartTime:     time.Now(),
		Delays:        []int{},
		LastHourReset: time.Now(),
		LastDayReset:  time.Now(),
	}

	messageTemplates = []string{
		"مرحباً {CustomerName}،\n\nنود أن نشكرك على كونك عميلاً مميزاً لدينا.\n\nرقم العميل: {Code}\n\nنتطلع لخدمتك دائماً.",
		"عزيزي {CustomerName}،\n\nنحن سعداء بخدمتك.\nكود العميل: {Code}\n\nشكراً لثقتك بنا.",
		"أهلاً {CustomerName}،\n\nنتمنى أن تكون بخير.\nرقمك لدينا: {Code}",
	}

	log                    *logger
	failedCustomers        []Customer
	selectedTemplates      []string // User-selected message templates
	templatePermutationIdx int      // Current template index for permutation
)

// displayError shows a professional error message with context and suggestions
func displayError(title, message, action string, suggestions []string) {
	fmt.Println()
	fmt.Println(bgRed + colorWhite + bold + " ✗ ERROR " + colorReset)
	fmt.Println(colorBrightRed + "┌─ " + title + colorReset)
	fmt.Println(colorRed + "│" + colorReset)
	fmt.Println(colorRed + "│  " + colorReset + message)
	fmt.Println(colorRed + "│" + colorReset)

	if action != "" {
		fmt.Println(colorRed + "├─ " + colorYellow + "What to do:" + colorReset)
		fmt.Println(colorRed + "│  " + colorReset + action)
		fmt.Println(colorRed + "│" + colorReset)
	}

	if len(suggestions) > 0 {
		fmt.Println(colorRed + "├─ " + colorCyan + "Suggestions:" + colorReset)
		for _, suggestion := range suggestions {
			fmt.Println(colorRed + "│  " + colorReset + dim + "• " + suggestion + colorReset)
		}
		fmt.Println(colorRed + "│" + colorReset)
	}

	fmt.Println(colorRed + "└─" + strings.Repeat("─", 58) + colorReset)
	fmt.Println()
}

// displayWarning shows a professional warning message
func displayWarning(title, message string, tips []string) {
	fmt.Println()
	fmt.Println(bgYellow + colorWhite + bold + " ⚠ WARNING " + colorReset)
	fmt.Println(colorYellow + "┌─ " + title + colorReset)
	fmt.Println(colorYellow + "│  " + colorReset + message)

	if len(tips) > 0 {
		fmt.Println(colorYellow + "│" + colorReset)
		fmt.Println(colorYellow + "├─ " + colorCyan + "Tips:" + colorReset)
		for _, tip := range tips {
			fmt.Println(colorYellow + "│  " + colorReset + dim + "• " + tip + colorReset)
		}
	}

	fmt.Println(colorYellow + "└─" + strings.Repeat("─", 58) + colorReset)
	fmt.Println()
}

// displaySuccess shows a professional success message
func displaySuccess(title, message string) {
	fmt.Println()
	fmt.Println(bgGreen + colorWhite + bold + " ✓ SUCCESS " + colorReset)
	fmt.Println(colorBrightGreen + "┌─ " + title + colorReset)
	fmt.Println(colorGreen + "│  " + colorReset + message)
	fmt.Println(colorGreen + "└─" + strings.Repeat("─", 58) + colorReset)
	fmt.Println()
}

// displayInfo shows a professional info message
func displayInfo(title, message string, details []string) {
	fmt.Println()
	fmt.Println(colorBrightCyan + "ℹ " + bold + title + colorReset)
	fmt.Println(colorCyan + "  " + colorReset + message)

	if len(details) > 0 {
		fmt.Println()
		for _, detail := range details {
			fmt.Println(colorCyan + "  • " + colorReset + dim + detail + colorReset)
		}
	}
	fmt.Println()
}

// displayProgress shows a professional progress indicator
func displayProgressBar(current, total int, label string) {
	percentage := float64(current) / float64(total) * 100
	barWidth := 40
	filled := int(float64(barWidth) * float64(current) / float64(total))

	bar := colorBrightGreen + strings.Repeat("█", filled) + colorReset +
		colorGray + strings.Repeat("░", barWidth-filled) + colorReset

	fmt.Printf("\r  %s [%s] %.1f%% (%d/%d)", label, bar, percentage, current, total)

	if current == total {
		fmt.Println() // New line when complete
	}
}

// loadTemplatesFromFiles reads all .txt and .md files in current directory
func loadTemplatesFromFiles() ([]string, error) {
	templates := make([]string, 0)
	templateFiles := make([]string, 0)

	// Read current directory
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, err
	}

	// Find all .txt and .md files
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") {
			templateFiles = append(templateFiles, name)
		}
	}

	// Also check templates/ directory if it exists
	if _, err := os.Stat("templates"); err == nil {
		templateDir, err := os.ReadDir("templates")
		if err == nil {
			for _, file := range templateDir {
				if file.IsDir() {
					continue
				}
				name := file.Name()
				if strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") {
					templateFiles = append(templateFiles, "templates/"+name)
				}
			}
		}
	}

	// Read content from each file
	for _, filename := range templateFiles {
		content, err := os.ReadFile(filename)
		if err != nil {
			log.Warning(fmt.Sprintf("Could not read template file %s: %v", filename, err))
			continue
		}

		// Skip empty files
		text := strings.TrimSpace(string(content))
		if text != "" {
			templates = append(templates, text)
			log.Info(fmt.Sprintf("Loaded template from: %s (%d chars)", filename, len(text)))
		}
	}

	return templates, nil
}

// selectTemplatesInteractive allows user to select which templates to use
func selectTemplatesInteractive(templates []string) ([]string, error) {
	if len(templates) == 0 {
		displayWarning("No Templates Found",
			"No .txt or .md template files found in current directory",
			[]string{
				"Using built-in default templates",
				"Create .txt or .md files with your message templates",
				"Place them in current directory or templates/ folder",
			})
		return messageTemplates, nil
	}

	fmt.Println(bold + colorBrightCyan + "\n📝 Message Templates Found" + colorReset)
	fmt.Println(colorCyan + strings.Repeat("─", 60) + colorReset)
	fmt.Println(dim + fmt.Sprintf("Found %d template files", len(templates)) + colorReset)
	fmt.Println()

	// Show preview of each template
	templatePreviews := make([]string, len(templates))
	for i, template := range templates {
		preview := template
		if len(preview) > 80 {
			preview = preview[:77] + "..."
		}
		// Replace newlines with space for preview
		preview = strings.ReplaceAll(preview, "\n", " ")
		templatePreviews[i] = fmt.Sprintf("Template %d: %s", i+1, preview)
	}

	// Add option to use all templates
	templatePreviews = append(templatePreviews, colorBrightGreen+"✓ Use ALL templates (Recommended)"+colorReset)
	templatePreviews = append(templatePreviews, colorYellow+"⚙ Use built-in default templates"+colorReset)

	prompt := promptui.Select{
		Label: "Select templates to use",
		Items: templatePreviews,
		Size:  10,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	// Use all templates
	if idx == len(templates) {
		displaySuccess("Templates Selected",
			fmt.Sprintf("Using all %d templates in permutation mode", len(templates)))
		return templates, nil
	}

	// Use built-in defaults
	if idx == len(templates)+1 {
		displayInfo("Using Defaults",
			"Using built-in Arabic message templates",
			[]string{
				fmt.Sprintf("%d default templates available", len(messageTemplates)),
			})
		return messageTemplates, nil
	}

	// Use single selected template
	displaySuccess("Template Selected",
		fmt.Sprintf("Using template %d", idx+1))
	return []string{templates[idx]}, nil
}

// getNextTemplateInPermutation returns the next template in rotation
func getNextTemplateInPermutation() string {
	if len(selectedTemplates) == 0 {
		selectedTemplates = messageTemplates
	}

	template := selectedTemplates[templatePermutationIdx]
	templatePermutationIdx = (templatePermutationIdx + 1) % len(selectedTemplates)
	return template
}

// displayWelcomeBanner displays a beautiful welcome banner
func displayWelcomeBanner() {
	// Clear screen for clean start
	fmt.Print("\033[H\033[2J")

	banner := `
╔══════════════════════════════════════════════════════════════╗
║                                                              ║
║     📱  WhatsApp Bulk Messaging System  📱                   ║
║                                                              ║
║     ` + dim + `Powered by Go + whatsmeow` + colorReset + colorCyan + `                                ║
║     ` + dim + `Version 1.0.0 - Production Ready` + colorReset + colorCyan + `                         ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Println(bold + colorBrightCyan + banner + colorReset)

	// Display safety notice
	fmt.Println(colorYellow + "⚠️  Important:" + colorReset + " This tool sends bulk messages. Use responsibly.")
	fmt.Println(dim + "   Follow WhatsApp's terms of service and local regulations." + colorReset)
	fmt.Println()
}

// configureInteractive prompts user for configuration
func configureInteractive() error {
	fmt.Println(bold + colorBrightCyan + "\n⚙️  Configuration Setup" + colorReset)
	fmt.Println(colorCyan + strings.Repeat("─", 60) + colorReset)
	fmt.Println(dim + "Let's configure your bulk messaging campaign" + colorReset)
	fmt.Println()

	// CSV File selection
	csvPrompt := promptui.Prompt{
		Label:   "CSV File Path",
		Default: "customers.csv",
	}
	csvFile, err := csvPrompt.Run()
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		fmt.Println()
		displayError("File Not Found",
			fmt.Sprintf("Cannot find CSV file: %s", csvFile),
			"Please check the file path and try again",
			[]string{
				"Ensure the file exists in the specified location",
				"Check for typos in the filename",
				"Use absolute path if relative path doesn't work",
			})
		return fmt.Errorf("CSV file not found")
	}

	// Show file info
	fileInfo, _ := os.Stat(csvFile)
	fmt.Printf(colorBrightGreen+"  ✓ CSV file found"+colorReset+": %s "+dim+"(%d bytes)"+colorReset+"\n",
		csvFile, fileInfo.Size())

	// Configuration mode selection
	modePrompt := promptui.Select{
		Label: "Configuration Mode",
		Items: []string{
			"Quick Start (Recommended defaults)",
			"Custom Configuration (Advanced)",
		},
	}
	modeIdx, _, err := modePrompt.Run()
	if err != nil {
		return err
	}

	if modeIdx == 1 {
		// Custom configuration
		if err := customConfiguration(); err != nil {
			return err
		}
	} else {
		// Use defaults
		fmt.Println(colorCyan + "\n✓ Using recommended defaults" + colorReset)
		displayCurrentConfig()
	}

	// Confirmation
	confirmPrompt := promptui.Select{
		Label: "Ready to start?",
		Items: []string{"Yes, start sending", "No, exit"},
	}
	confirmIdx, _, err := confirmPrompt.Run()
	if err != nil {
		return err
	}

	if confirmIdx != 0 {
		return fmt.Errorf("user cancelled")
	}

	fmt.Println()
	return nil
}

// customConfiguration allows user to customize settings
func customConfiguration() error {
	fmt.Println(colorYellow + "\n🔧 Custom Configuration" + colorReset)

	// Delay between messages
	delayPrompt := promptui.Prompt{
		Label:   "Delay between messages (seconds, min-max)",
		Default: "5-12",
		Validate: func(input string) error {
			parts := strings.Split(input, "-")
			if len(parts) != 2 {
				return fmt.Errorf("format should be: min-max (e.g., 5-12)")
			}
			return nil
		},
	}
	delayStr, err := delayPrompt.Run()
	if err != nil {
		return err
	}
	parts := strings.Split(delayStr, "-")
	config.DelayMin, _ = strconv.Atoi(parts[0])
	config.DelayMax, _ = strconv.Atoi(parts[1])
	config.DelayMin *= 1000
	config.DelayMax *= 1000

	// Batch size
	batchPrompt := promptui.Prompt{
		Label:   "Messages per batch",
		Default: "20",
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil || val < 1 || val > 50 {
				return fmt.Errorf("must be between 1 and 50")
			}
			return nil
		},
	}
	batchStr, err := batchPrompt.Run()
	if err != nil {
		return err
	}
	config.BatchSize, _ = strconv.Atoi(batchStr)

	// Batch delay
	batchDelayPrompt := promptui.Prompt{
		Label:   "Break between batches (seconds)",
		Default: "120",
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil || val < 30 {
				return fmt.Errorf("must be at least 30 seconds")
			}
			return nil
		},
	}
	batchDelayStr, err := batchDelayPrompt.Run()
	if err != nil {
		return err
	}
	delay, _ := strconv.Atoi(batchDelayStr)
	config.BatchDelay = delay * 1000

	// Skip duplicates
	skipDupPrompt := promptui.Select{
		Label: "Skip duplicate phone numbers?",
		Items: []string{"Yes (Recommended)", "No"},
	}
	skipDupIdx, _, err := skipDupPrompt.Run()
	if err != nil {
		return err
	}
	config.SkipDuplicates = (skipDupIdx == 0)

	// Max retries
	retryPrompt := promptui.Prompt{
		Label:   "Max retry attempts per message",
		Default: "3",
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil || val < 0 || val > 5 {
				return fmt.Errorf("must be between 0 and 5")
			}
			return nil
		},
	}
	retryStr, err := retryPrompt.Run()
	if err != nil {
		return err
	}
	config.MaxRetries, _ = strconv.Atoi(retryStr)

	fmt.Println(colorGreen + "\n✓ Configuration updated" + colorReset)
	displayCurrentConfig()

	return nil
}

// displayCurrentConfig shows current configuration
func displayCurrentConfig() {
	fmt.Println(colorCyan + "\n📋 Current Configuration:" + colorReset)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  Delay Between Messages:  %d-%d seconds\n", config.DelayMin/1000, config.DelayMax/1000)
	fmt.Printf("  Batch Size:              %d messages\n", config.BatchSize)
	fmt.Printf("  Batch Break:             %d seconds\n", config.BatchDelay/1000)
	fmt.Printf("  Max Retries:             %d attempts\n", config.MaxRetries)
	fmt.Printf("  Skip Duplicates:         %v\n", config.SkipDuplicates)
	fmt.Printf("  Skip Invalid Numbers:    %v\n", config.SkipInvalid)
	fmt.Printf("  Pre-Check Numbers:       %v\n", config.PreCheckNumbers)
	fmt.Printf("  Country Code:            +%s\n", config.CountryCode)
	fmt.Println(strings.Repeat("─", 60))
}

func main() {
	// Initialize logger
	log = NewLogger()

	// Display welcome banner
	displayWelcomeBanner()

	// Load message templates from files
	log.Info("Scanning for message templates...")
	fileTemplates, err := loadTemplatesFromFiles()
	if err != nil {
		log.Warning(fmt.Sprintf("Could not scan templates: %v", err))
	}

	// Let user select templates
	selectedTemplates, err = selectTemplatesInteractive(fileTemplates)
	if err != nil {
		log.Error("Template selection failed", err)
		return
	}

	// Show template info
	displayInfo("Template Configuration",
		fmt.Sprintf("Using %d message template(s) in permutation mode", len(selectedTemplates)),
		[]string{
			"Each customer will receive a different template",
			"Templates rotate automatically",
			"Ensures message variety",
		})

	// Interactive configuration
	if err := configureInteractive(); err != nil {
		log.Error("Configuration failed", err)
		return
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Warning("Shutdown signal received, cleaning up...")
		cancel()
	}()

	// Load CSV
	customers, err := loadCSV("customers.csv")
	if err != nil {
		log.Error("Failed to load CSV", err)
		return
	}

	if len(customers) == 0 {
		log.Error("No customers found in CSV", nil)
		return
	}

	log.Info(fmt.Sprintf("Loaded %d customers from CSV", len(customers)))

	// Process and validate customers
	processedCustomers := processCustomers(customers)
	if len(processedCustomers) == 0 {
		log.Error("No valid customers to process", nil)
		return
	}

	log.Info(fmt.Sprintf("Valid customers ready: %d", len(processedCustomers)))

	// Display execution plan
	displayExecutionPlan(len(processedCustomers))

	// Preview first message
	if len(processedCustomers) > 0 {
		previewMessage(processedCustomers[0])
	}

	// Wait before starting
	log.Info("Starting in 5 seconds...")
	time.Sleep(5 * time.Second)

	// Initialize WhatsApp client
	client, err := initializeWhatsApp(ctx)
	if err != nil {
		log.Error("Failed to initialize WhatsApp", err)
		return
	}
	defer client.Disconnect()

	// Pre-check numbers if enabled
	if config.PreCheckNumbers {
		log.Info("Pre-checking all numbers on WhatsApp...")
		customers = preCheckWhatsAppNumbers(ctx, client, customers)

		// Save updated CSV with has_whatsapp column
		if err := saveCustomersWithWhatsAppStatus(customers); err != nil {
			log.Error("Failed to save updated CSV", err)
		} else {
			log.Success("Updated CSV saved with WhatsApp status")
		}

		// Re-process customers after pre-check
		processedCustomers = processCustomers(customers)
		log.Info(fmt.Sprintf("After pre-check: %d valid customers", len(processedCustomers)))
	}

	// Initialize progress
	progress.Total = len(processedCustomers)

	// Send messages
	sendMessagesToCustomers(ctx, client, processedCustomers)

	// Generate report
	generateReport()

	// Save failed customers
	if config.SaveFailed && len(failedCustomers) > 0 {
		saveFailedCustomers(failedCustomers)
	}

	log.Success("Bulk messaging completed")
}

// initializeWhatsApp initializes the WhatsApp client
func initializeWhatsApp(ctx context.Context) (*whatsmeow.Client, error) {
	log.Info("Initializing WhatsApp client...")

	// Setup database for session storage
	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New(ctx, "sqlite3", "file:whatsapp_session.db?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, err
	}

	// Get first device or create new
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Register event handlers
	client.AddEventHandler(func(evt interface{}) {
		// Handle events if needed
	})

	// Connect
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(ctx)
		err = client.Connect()
		if err != nil {
			return nil, err
		}

		log.Info("Scan QR code with WhatsApp:")
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				log.Info(fmt.Sprintf("QR channel result: %s", evt.Event))
			}
		}
	} else {
		// Already logged in
		err = client.Connect()
		if err != nil {
			return nil, err
		}
	}

	log.Success("WhatsApp client connected successfully")
	return client, nil
}

// preCheckWhatsAppNumbers checks all numbers on WhatsApp and updates the HasWhatsApp field
func preCheckWhatsAppNumbers(ctx context.Context, client *whatsmeow.Client, customers []Customer) []Customer {
	total := len(customers)
	onWhatsApp := 0
	notOnWhatsApp := 0
	alreadyChecked := 0
	batchSize := 50 // Check 50 numbers at a time

	fmt.Println(colorCyan + "\n🔍 Checking WhatsApp Status (Batch Mode)..." + colorReset)
	fmt.Println(strings.Repeat("─", 60))

	// Prepare batch data
	type checkItem struct {
		index     int
		phone     string
		formatted string
	}

	toCheck := []checkItem{}

	// First pass: collect numbers to check and validate format
	for i := range customers {
		// Skip if already checked
		if customers[i].HasWhatsApp != "" {
			alreadyChecked++
			continue
		}

		// Get phone number
		phone := customers[i].Mobile
		if phone == "" {
			phone = customers[i].Phone
		}

		// Clean and format
		cleaned := cleanPhoneNumber(phone)
		formatted := formatPhoneNumber(cleaned)

		// Validate format
		if len(formatted) != config.PhoneLength {
			customers[i].HasWhatsApp = "no"
			notOnWhatsApp++
			continue
		}

		toCheck = append(toCheck, checkItem{
			index:     i,
			phone:     phone,
			formatted: formatted,
		})
	}

	// Process in batches
	totalBatches := (len(toCheck) + batchSize - 1) / batchSize

	for batchNum := 0; batchNum < totalBatches; batchNum++ {
		start := batchNum * batchSize
		end := start + batchSize
		if end > len(toCheck) {
			end = len(toCheck)
		}

		batch := toCheck[start:end]

		// Prepare phone list for batch check
		phoneList := make([]string, len(batch))
		for i, item := range batch {
			phoneList[i] = item.formatted
		}

		// Batch check on WhatsApp
		exists, err := client.IsOnWhatsApp(phoneList)
		if err != nil {
			log.Warning(fmt.Sprintf("Batch check failed: %v", err))
			// Mark all in batch as unchecked on error
			time.Sleep(time.Duration(config.CheckDelay) * time.Millisecond)
			continue
		}

		// Update results
		for i, item := range batch {
			if i < len(exists) && exists[i].IsIn {
				customers[item.index].HasWhatsApp = "yes"
				onWhatsApp++
			} else {
				customers[item.index].HasWhatsApp = "no"
				notOnWhatsApp++
			}
		}

		// Display progress
		checked := end
		percentage := float64(checked+alreadyChecked) / float64(total) * 100
		fmt.Printf("\r  Progress: %.1f%% (%d/%d) - ✓ %d  ✗ %d  ⊙ %d  [Batch %d/%d]",
			percentage, checked+alreadyChecked, total, onWhatsApp, notOnWhatsApp, alreadyChecked, batchNum+1, totalBatches)

		// Delay between batches to avoid rate limiting
		if batchNum < totalBatches-1 {
			time.Sleep(time.Duration(config.CheckDelay) * time.Millisecond)
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			fmt.Println("\n" + colorYellow + "Check cancelled by user" + colorReset)
			return customers
		default:
		}
	}

	fmt.Println() // New line after progress
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf(colorGreen+"✓ Check complete: %d on WhatsApp, %d not on WhatsApp, %d already checked\n"+colorReset,
		onWhatsApp, notOnWhatsApp, alreadyChecked)
	fmt.Printf(colorCyan+"  Checked in %d batches of up to %d numbers\n"+colorReset, totalBatches, batchSize)

	return customers
}

// saveCustomersWithWhatsAppStatus saves customers CSV with has_whatsapp column
func saveCustomersWithWhatsAppStatus(customers []Customer) error {
	// Create data directory if it doesn't exist
	os.MkdirAll("data", 0755)

	// Create new CSV file
	file, err := os.Create("data/customers_checked.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header with has_whatsapp column
	writer.Write([]string{"Code", "CustomerName", "Phone", "Mobile", "HasWhatsApp"})

	// Write customers
	for _, c := range customers {
		hasWhatsApp := c.HasWhatsApp
		if hasWhatsApp == "" {
			hasWhatsApp = "unchecked"
		}
		writer.Write([]string{c.Code, c.CustomerName, c.Phone, c.Mobile, hasWhatsApp})
	}

	return nil
}

// loadCSV loads customers from CSV file
func loadCSV(filename string) ([]Customer, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or has no data rows")
	}

	// Check if HasWhatsApp column exists
	hasWhatsAppCol := false
	if len(records[0]) >= 5 {
		header := strings.ToLower(strings.TrimSpace(records[0][4]))
		if header == "haswhatsapp" || header == "has_whatsapp" {
			hasWhatsAppCol = true
		}
	}

	// Parse customers (skip header)
	customers := make([]Customer, 0)
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 4 {
			continue
		}

		customer := Customer{
			Code:         strings.TrimSpace(records[i][0]),
			CustomerName: strings.TrimSpace(records[i][1]),
			Phone:        strings.TrimSpace(records[i][2]),
			Mobile:       strings.TrimSpace(records[i][3]),
		}

		// Load HasWhatsApp status if column exists
		if hasWhatsAppCol && len(records[i]) >= 5 {
			customer.HasWhatsApp = strings.ToLower(strings.TrimSpace(records[i][4]))
		}

		customers = append(customers, customer)
	}

	return customers, nil
}

// processCustomers validates and processes customers
func processCustomers(customers []Customer) []ProcessedCustomer {
	processed := make([]ProcessedCustomer, 0)
	seenPhones := make(map[string]bool) // Track seen phone numbers to avoid duplicates

	for _, customer := range customers {
		// Skip if already checked and not on WhatsApp
		if customer.HasWhatsApp == "no" {
			log.Warning(fmt.Sprintf("Skipping %s - Not on WhatsApp (pre-checked)", customer.CustomerName))
			progress.Skipped++
			continue
		}

		// Skip special entries
		if shouldSkipCustomer(customer) {
			log.Warning(fmt.Sprintf("Skipping customer: %s", customer.CustomerName))
			progress.Skipped++
			continue
		}

		// Validate customer data
		if !validateCustomerData(customer) {
			log.Warning(fmt.Sprintf("Invalid customer data: %s", customer.CustomerName))
			progress.Skipped++
			continue
		}

		// Select best phone number
		selectedPhone := selectBestPhone(customer)

		// Validate and format phone
		formattedPhone, isValid, validationError := validateAndFormatPhone(selectedPhone)

		pc := ProcessedCustomer{
			Customer:        customer,
			SelectedPhone:   selectedPhone,
			FormattedPhone:  formattedPhone,
			IsValid:         isValid,
			ValidationError: validationError,
		}

		if !isValid && config.SkipInvalid {
			log.Warning(fmt.Sprintf("Skipping %s - Invalid phone: %s", customer.CustomerName, validationError))
			progress.Skipped++
			continue
		}

		// Check for duplicate phone numbers (if enabled)
		if config.SkipDuplicates {
			if seenPhones[formattedPhone] {
				log.Warning(fmt.Sprintf("Skipping %s - Duplicate phone number: %s", customer.CustomerName, formattedPhone))
				progress.Skipped++
				progress.Duplicates++
				continue
			}

			// Mark phone as seen
			seenPhones[formattedPhone] = true
		}

		processed = append(processed, pc)
	}

	return processed
}

// shouldSkipCustomer checks if customer should be skipped
func shouldSkipCustomer(customer Customer) bool {
	upperName := strings.ToUpper(customer.CustomerName)
	return strings.Contains(upperName, "SPECIAL ORDER") || customer.Code == "0"
}

// validateCustomerData validates customer data
func validateCustomerData(customer Customer) bool {
	if customer.CustomerName == "" || customer.Code == "" {
		return false
	}
	if customer.Phone == "" && customer.Mobile == "" {
		return false
	}
	return true
}

// selectBestPhone selects the best phone number
func selectBestPhone(customer Customer) string {
	if config.PreferMobile && customer.Mobile != "" {
		return customer.Mobile
	}
	if customer.Phone != "" {
		return customer.Phone
	}
	return customer.Mobile
}

// validateAndFormatPhone validates and formats phone number
func validateAndFormatPhone(phone string) (string, bool, string) {
	if phone == "" {
		return "", false, "Phone number is empty"
	}

	// Check for invalid patterns
	invalidPatterns := []string{"*****", "***", "65354", "10000000"}
	for _, pattern := range invalidPatterns {
		if phone == pattern {
			return "", false, "Invalid phone number pattern"
		}
	}

	// Clean phone number
	cleaned := cleanPhoneNumber(phone)
	if len(cleaned) == 0 {
		return "", false, "No digits in phone number"
	}

	// Format phone number
	formatted := formatPhoneNumber(cleaned)

	// Validate length
	if len(formatted) != config.PhoneLength {
		return "", false, fmt.Sprintf("Invalid length: %d, expected %d", len(formatted), config.PhoneLength)
	}

	// Validate country code
	if !strings.HasPrefix(formatted, config.CountryCode) {
		return "", false, fmt.Sprintf("Must start with %s", config.CountryCode)
	}

	return formatted, true, ""
}

// cleanPhoneNumber removes non-digit characters
func cleanPhoneNumber(phone string) string {
	result := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			result += string(char)
		}
	}
	return result
}

// formatPhoneNumber formats phone to WhatsApp format
func formatPhoneNumber(phone string) string {
	// Remove leading 0
	phone = strings.TrimPrefix(phone, "0")

	// Add country code if missing
	if !strings.HasPrefix(phone, config.CountryCode) {
		phone = config.CountryCode + phone
	}

	return phone
}

// sendMessagesToCustomers sends messages to all customers
func sendMessagesToCustomers(ctx context.Context, client *whatsmeow.Client, customers []ProcessedCustomer) {
	log.Info(fmt.Sprintf("Starting to send messages to %d customers", len(customers)))

	for i, customer := range customers {
		select {
		case <-ctx.Done():
			log.Warning("Shutdown requested, stopping")
			return
		default:
		}

		isWarmup := i < 5

		// Display progress
		displayProgress(i+1, len(customers), customer.CustomerName)

		// Send message with retry
		result := sendMessageWithRetry(client, customer, isWarmup)

		// Calculate delay
		delay := getRandomDelay(isWarmup)
		progress.Delays = append(progress.Delays, delay)

		// Record result
		recordResult(result)

		// Check for batch break
		if shouldTakeBatchBreak(i + 1) {
			clearProgress()
			log.Info(fmt.Sprintf("Batch completed. Taking %d second break...", config.BatchDelay/1000))
			displayStats()
			time.Sleep(time.Duration(config.BatchDelay) * time.Millisecond)
			log.Info("Resuming...")
		} else {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
	}

	clearProgress()
	log.Success("All messages processed")
}

// sendMessageWithRetry sends message with retry logic
func sendMessageWithRetry(client *whatsmeow.Client, customer ProcessedCustomer, isWarmup bool) MessageResult {
	var lastError string

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Render message
		message := renderMessage(customer)

		// Format WhatsApp JID
		jid := types.NewJID(customer.FormattedPhone, types.DefaultUserServer)

		// Send message directly (WhatsApp will return error if number doesn't exist)
		_, err := client.SendMessage(context.Background(), jid, &waE2E.Message{
			Conversation: proto.String(message),
		})

		if err != nil {
			lastError = err.Error()
			if attempt < config.MaxRetries {
				log.Warning(fmt.Sprintf("Attempt %d failed for %s, retrying...", attempt+1, customer.CustomerName))
				time.Sleep(time.Duration(config.RetryDelay) * time.Millisecond)
				continue
			}
		} else {
			return MessageResult{
				Customer:   customer,
				Success:    true,
				Timestamp:  time.Now(),
				RetryCount: attempt,
			}
		}
	}

	return MessageResult{
		Customer:   customer,
		Success:    false,
		Timestamp:  time.Now(),
		Error:      lastError,
		RetryCount: config.MaxRetries,
	}
}

// renderMessage renders message template using permutation
func renderMessage(customer ProcessedCustomer) string {
	// Get next template in permutation order
	template := getNextTemplateInPermutation()

	// Replace placeholders
	message := template
	message = strings.ReplaceAll(message, "{CustomerName}", customer.CustomerName)
	message = strings.ReplaceAll(message, "{Code}", customer.Code)
	message = strings.ReplaceAll(message, "{Phone}", customer.Phone)
	message = strings.ReplaceAll(message, "{Mobile}", customer.Mobile)

	return message
}

// getRandomDelay returns random delay with anti-blocking enhancements
func getRandomDelay(isWarmup bool) int {
	if isWarmup {
		return config.WarmupDelay
	}
	
	// Base delay
	baseDelay := config.DelayMin + rand.Intn(config.DelayMax-config.DelayMin+1)
	
	// Add micro-jitter if enabled (±0.5-2 seconds)
	if config.AddJitter {
		jitter := rand.Intn(2000) - 500 // -500ms to +1500ms
		baseDelay += jitter
	}
	
	// Occasional long pause (default 5% chance)
	if rand.Float32() < config.LongPauseChance {
		longPause := 30000 + rand.Intn(30000) // 30-60 seconds
		log.Info(fmt.Sprintf("Taking extended pause: %d seconds", longPause/1000))
		return baseDelay + longPause
	}
	
	return baseDelay
}

// isBusinessHours checks if current time is within business hours
func isBusinessHours() bool {
	if !config.BusinessHoursOnly {
		return true // No restriction
	}
	
	now := time.Now()
	hour := now.Hour()
	
	// Business hours: 9 AM to 9 PM
	if hour < 9 || hour >= 21 {
		return false
	}
	
	return true
}

// checkRateLimits checks if we can send more messages
func checkRateLimits() (bool, string) {
	now := time.Now()
	
	// Reset hourly counter if needed
	if now.Sub(progress.LastHourReset) >= time.Hour {
		progress.HourlySent = 0
		progress.LastHourReset = now
	}
	
	// Reset daily counter if needed
	if now.Sub(progress.LastDayReset) >= 24*time.Hour {
		progress.DailySent = 0
		progress.LastDayReset = now
	}
	
	// Check hourly limit
	if progress.HourlySent >= config.HourlyLimit {
		minutesLeft := 60 - int(now.Sub(progress.LastHourReset).Minutes())
		return false, fmt.Sprintf("Hourly limit reached (%d/%d). Wait %d minutes.", 
			progress.HourlySent, config.HourlyLimit, minutesLeft)
	}
	
	// Check daily limit
	if progress.DailySent >= config.DailyLimit {
		hoursLeft := 24 - int(now.Sub(progress.LastDayReset).Hours())
		return false, fmt.Sprintf("Daily limit reached (%d/%d). Wait %d hours.", 
			progress.DailySent, config.DailyLimit, hoursLeft)
	}
	
	return true, ""
}

// incrementRateLimiters increments the rate limit counters
func incrementRateLimiters() {
	progress.HourlySent++
	progress.DailySent++
}

// simulateTypingDelay calculates and applies typing delay based on message length
func simulateTypingDelay(message string) {
	if !config.SimulateTyping {
		return
	}
	
	// Calculate typing time (40-60 characters per second)
	charsPerSecond := 40 + rand.Intn(20)
	typingTimeMs := (len(message) * 1000) / charsPerSecond
	
	// Add some randomness (±20%)
	variation := int(float64(typingTimeMs) * 0.2)
	typingTimeMs += rand.Intn(variation*2) - variation
	
	// Minimum 1 second, maximum 10 seconds
	if typingTimeMs < 1000 {
		typingTimeMs = 1000
	}
	if typingTimeMs > 10000 {
		typingTimeMs = 10000
	}
	
	time.Sleep(time.Duration(typingTimeMs) * time.Millisecond)
}

// shouldTakeBatchBreak checks if batch break is needed
func shouldTakeBatchBreak(count int) bool {
	return count > 0 && count%config.BatchSize == 0
}

// recordResult records message result
func recordResult(result MessageResult) {
	progress.Processed++
	if result.Success {
		progress.Successful++
		log.Success(fmt.Sprintf("Message sent to %s (%s)", result.Customer.CustomerName, result.Customer.FormattedPhone))
	} else {
		progress.Failed++
		failedCustomers = append(failedCustomers, result.Customer.Customer)
		log.Error(fmt.Sprintf("Failed to send to %s: %s", result.Customer.CustomerName, result.Error), nil)
	}
}

// Helper functions for display
func displayExecutionPlan(count int) {
	avgDelay := (config.DelayMin + config.DelayMax) / 2
	batchCount := (count + config.BatchSize - 1) / config.BatchSize
	totalMs := count*avgDelay + (batchCount-1)*config.BatchDelay
	minutes := totalMs / 60000
	seconds := (totalMs % 60000) / 1000

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXECUTION PLAN")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Customers:        %d\n", count)
	fmt.Printf("Batch Size:             %d messages\n", config.BatchSize)
	fmt.Printf("Number of Batches:      %d\n", batchCount)
	fmt.Printf("Delay Between Messages: %d-%ds\n", config.DelayMin/1000, config.DelayMax/1000)
	fmt.Printf("Delay Between Batches:  %ds\n", config.BatchDelay/1000)
	fmt.Printf("Estimated Duration:     %dm %ds\n", minutes, seconds)
	fmt.Printf("Max Retries:            %d\n", config.MaxRetries)
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

func previewMessage(customer ProcessedCustomer) {
	message := renderMessage(customer)
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("MESSAGE PREVIEW")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("To: %s\n", customer.CustomerName)
	fmt.Printf("Phone: %s\n", customer.FormattedPhone)
	fmt.Printf("Length: %d characters\n", len(message))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(message)
	fmt.Println(strings.Repeat("─", 60) + "\n")
}

func displayProgress(current, total int, name string) {
	percentage := float64(current) / float64(total) * 100
	barLength := 30
	filled := int(float64(barLength) * float64(current) / float64(total))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barLength-filled)

	if len(name) > 30 {
		name = name[:30]
	}

	fmt.Printf("\r[%s] %.1f%% (%d/%d) - Processing: %-30s", bar, percentage, current, total, name)
}

func clearProgress() {
	fmt.Print("\r" + strings.Repeat(" ", 120) + "\r")
}

func displayStats() {
	successRate := 0.0
	if progress.Successful+progress.Failed > 0 {
		successRate = float64(progress.Successful) / float64(progress.Successful+progress.Failed) * 100
	}

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("CURRENT STATISTICS")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("Processed:     %d/%d\n", progress.Processed, progress.Total)
	fmt.Printf("Successful:    %d\n", progress.Successful)
	fmt.Printf("Failed:        %d\n", progress.Failed)
	fmt.Printf("Skipped:       %d\n", progress.Skipped)
	if progress.Duplicates > 0 {
		fmt.Printf("  - Duplicates: %d\n", progress.Duplicates)
	}
	fmt.Printf("Success Rate:  %.2f%%\n", successRate)
	fmt.Println(strings.Repeat("─", 60) + "\n")
}

func generateReport() {
	duration := time.Since(progress.StartTime)
	successRate := 0.0
	if progress.Successful+progress.Failed > 0 {
		successRate = float64(progress.Successful) / float64(progress.Successful+progress.Failed) * 100
	}

	avgDelay := 0
	if len(progress.Delays) > 0 {
		sum := 0
		for _, d := range progress.Delays {
			sum += d
		}
		avgDelay = sum / len(progress.Delays)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("EXECUTION SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Start Time:         %s\n", progress.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("End Time:           %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration:           %s\n", duration.Round(time.Second))
	fmt.Printf("Total Customers:    %d\n", progress.Total)
	fmt.Printf("Successful Sends:   %d (%.2f%%)\n", progress.Successful, successRate)
	fmt.Printf("Failed Sends:       %d\n", progress.Failed)
	fmt.Printf("Skipped Customers:  %d\n", progress.Skipped)
	if progress.Duplicates > 0 {
		fmt.Printf("  - Duplicates:     %d\n", progress.Duplicates)
	}
	fmt.Printf("Average Delay:      %.2fs\n", float64(avgDelay)/1000)
	fmt.Println(strings.Repeat("=", 60) + "\n")
}

func saveFailedCustomers(customers []Customer) {
	file, err := os.Create("data/failed-customers.csv")
	if err != nil {
		log.Error("Failed to create failed customers file", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Code", "CustomerName", "Phone", "Mobile"})

	// Write customers
	for _, c := range customers {
		writer.Write([]string{c.Code, c.CustomerName, c.Phone, c.Mobile})
	}

	log.Info(fmt.Sprintf("Saved %d failed customers to data/failed-customers.csv", len(customers)))
}
