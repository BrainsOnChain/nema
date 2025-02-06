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

	"github.com/brainsonchain/nema/mock"
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
		log.Fatal("error creating logger")
	}
	defer logger.Sync()
	logger.Info("logger created")

	ctx := context.Background()
	if err := run(ctx, logger); err != nil {
		logger.Error("error running", zap.Error(err))
	}
}

func run(ctx context.Context, l *zap.Logger) error {
	// -------------------------------------------------------------------------
	// ENV VARS
	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env file")
	}
	l.Info("env vars loaded")

	// // -------------------------------------------------------------------------
	// // Prometheus Metrics
	// l.Info("initializing prometheus metrics")

	// http.Handle("/metrics", promhttp.Handler())
	// go http.ListenAndServe(":9091", nil)

	// -------------------------------------------------------------------------
	// DBM
	l.Info("creating dbm")

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "nema.db"
	}

	db, err := nema.NewDBManager(dbPath)
	if err != nil {
		return fmt.Errorf("error creating DBM: %w", err)
	}
	if err := db.Initiate(); err != nil {
		return fmt.Errorf("error initiating DBM: %w", err)
	}

	// -------------------------------------------------------------------------
	// Initial Prompt
	l.Info("reading initial prompt")

	initialPromptBytes, err := nemaPrompt.ReadFile("nema_prompt.txt")
	if err != nil {
		return fmt.Errorf("error reading prompt file: %w", err)
	}
	initialPrompt := string(initialPromptBytes)

	// -------------------------------------------------------------------------
	// LLM
	l.Info("creating llm")

	// Check the MODEL_PROVIDER env var. If ollama is set, use the ollama client
	// to create the LLM. Otherwise, use the openai client.
	var llm llms.Model

	switch os.Getenv("MODEL_PROVIDER") {
	case "ollama":
		l.Info("creating ollama client")
		llm, err = ollama.New(ollama.WithModel(os.Getenv("OLLAMA_MODEL")))
		if err != nil {
			return fmt.Errorf("error creating ollama client: %w", err)
		}
	case "openai":
		l.Info("creating openai client")
		llm, err = openai.New()
		if err != nil {
			return fmt.Errorf("error creating LLM: %w", err)
		}
	default:
		l.Info("creating mock llm")
		llm = &mock.MockLLM{}
	}

	// -------------------------------------------------------------------------
	// Nema
	l.Info("creating nema manager")

	nemaManager, err := nema.NewManager(l, db, initialPrompt, llm)
	if err != nil {
		return fmt.Errorf("error creating Nema Manager: %w", err)
	}

	// -------------------------------------------------------------------------
	// SERVER
	l.Info("creating server")

	srv := server.NewServer(l, nemaManager)

	// -------------------------------------------------------------------------
	// ERROR CHANNEL
	l.Info("creating error channel")

	errChan := make(chan error)

	// Run the server on port 8080
	go func() {
		l.Info("starting servers on port 8080 and 8081")
		if err := srv.Start(ctx, "8080", "8081"); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for any server errors
	if err := <-errChan; err != nil {
		l.Error("server error", zap.Error(err))
		return err
	}

	return nil
}
