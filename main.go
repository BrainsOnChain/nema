package main

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/brainsonchain/nema/dbm"
	"github.com/brainsonchain/nema/server"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

//go:embed nema_prompt.txt
var nemaPrompt embed.FS

func main() {
	// -------------------------------------------------------------------------
	// ENV VARS
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// -------------------------------------------------------------------------
	// LOGGER
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := config.Build()
	if err != nil {
		log.Fatal("Error creating logger")
	}
	defer logger.Sync()
	logger.Info("Logger created")

	ctx := context.Background()
	if err := run(ctx, logger); err != nil {
		logger.Error("Error running", zap.Error(err))
	}
}

func run(ctx context.Context, l *zap.Logger) error {
	// -------------------------------------------------------------------------
	// DBM
	l.Info("Creating DBM")
	db := dbm.NewManager()

	// -------------------------------------------------------------------------
	// SERVER
	l.Info("Creating server")
	server := server.NewServer(l, db)

	// -------------------------------------------------------------------------
	// ERROR CHANNEL
	errChan := make(chan error)

	// Run the server on port 8080
	go func() {
		l.Info("Starting server on port 8080")
		if err := server.Start(ctx, "8080"); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// -------------------------------------------------------------------------
	// Run Nema

	// Prompt user if they want to run Nema
	fmt.Println("Would you like to run Nema? (y/n)")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}
	response = strings.TrimSpace(response)
	if response != "y" {
		l.Info("Nema not run")
	} else {
		l.Info("Running Nema")
		runNema(ctx)
	}

	// Wait for any server errors
	return <-errChan
}

func runNema(ctx context.Context) error {
	// Read the embedded prompt file
	promptBytes, err := nemaPrompt.ReadFile("nema_prompt.txt")
	if err != nil {
		return fmt.Errorf("error reading prompt file: %w", err)
	}
	initialPrompt := string(promptBytes)

	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, initialPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, "What would you like to do?"),
	}

	llm, err := openai.New()
	if err != nil {
		return fmt.Errorf("error creating LLM: %w", err)
	}

	// Get initial response
	completion, err := llm.GenerateContent(ctx, messages, llms.WithTemperature(1))
	if err != nil {
		return fmt.Errorf("error generating completion: %w", err)
	}

	fmt.Println("\n🤖 Assistant:", completion.Choices[0].Content)

	// Start interactive loop
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n👤 You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		// Trim whitespace and check for exit command
		input = strings.TrimSpace(input)
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye! 👋")
			return nil
		}

		messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, input))

		fmt.Println("message length: ", len(messages))

		fmt.Print("\n🪱 Nema: ")
		completion, err := llm.GenerateContent(
			ctx,
			messages,
			llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
				fmt.Print(string(chunk))
				return nil
			}),
			llms.WithTemperature(1),
		)
		if err != nil {
			return fmt.Errorf("error streaming completion: %w", err)
		}

		content := completion.Choices[0].Content
		messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, content))
	}
}
