package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/brainsonchain/nema/nema"
	"github.com/brainsonchain/nema/server"
)

//go:embed nema_prompt.txt
var nemaPrompt embed.FS

func main() {

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
	// ENV VARS
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	l.Info("ENV VARS loaded")

	// -------------------------------------------------------------------------
	// DBM
	l.Info("Creating DBM")

	db, err := nema.NewDBManager("nema.db")
	if err != nil {
		return fmt.Errorf("error creating DBM: %w", err)
	}
	if err := db.Initiate(); err != nil {
		return fmt.Errorf("error initiating DBM: %w", err)
	}

	// -------------------------------------------------------------------------
	// Initial Prompt
	l.Info("Reading initial prompt")

	initialPromptBytes, err := nemaPrompt.ReadFile("nema_prompt.txt")
	if err != nil {
		return fmt.Errorf("error reading prompt file: %w", err)
	}
	initialPrompt := string(initialPromptBytes)

	// -------------------------------------------------------------------------
	// LLM
	l.Info("Creating LLM")

	// Check the MODEL_PROVIDER env var. If ollama is set, use the ollama client
	// to create the LLM. Otherwise, use the openai client.
	var llm llms.Model
	modelProvider := os.Getenv("MODEL_PROVIDER")
	if modelProvider == "ollama" {
		l.Info("Creating ollama client")
		ollama, err := ollama.New(
			ollama.WithModel(os.Getenv("OLLAMA_MODEL")),
		)
		if err != nil {
			return fmt.Errorf("error creating ollama client: %w", err)
		}
		llm = ollama
	} else {
		l.Info("Creating openai client")
		llm, err = openai.New()
		if err != nil {
			return fmt.Errorf("error creating LLM: %w", err)
		}
	}

	// -------------------------------------------------------------------------
	// Nema
	l.Info("Creating Nema Manager")

	nemaManager, err := nema.NewManager(l, db, initialPrompt, llm)
	if err != nil {
		return fmt.Errorf("error creating Nema Manager: %w", err)
	}

	// -------------------------------------------------------------------------
	// SERVER
	l.Info("Creating server")

	server := server.NewServer(l, nemaManager)

	// -------------------------------------------------------------------------
	// ERROR CHANNEL
	l.Info("Creating error channel")

	errChan := make(chan error)

	// Run the server on port 8080
	go func() {
		l.Info("Starting server on port 8080")
		if err := server.Start(ctx, "8080"); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for any server errors
	return <-errChan
}
