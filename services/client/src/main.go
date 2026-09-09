package main

import (
	"errors"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/client"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
)

func loadConfig() (client.Config, error) {
	agencyId := os.Getenv("AGENCY_ID")
	if agencyId == "" {
		return client.Config{}, errors.New("AGENCY_ID environment variable is required")
	}

	serverHost := os.Getenv("SERVER_HOST")
	if serverHost == "" {
		return client.Config{}, errors.New("SERVER_HOST environment variable is required")
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		return client.Config{}, errors.New("SERVER_PORT environment variable is required")
	}

	inputFile := os.Getenv("INPUT_FILE")
	if inputFile == "" {
		return client.Config{}, errors.New("INPUT_FILE environment variable is required")
	}

	outputFile := os.Getenv("OUTPUT_FILE")
	if outputFile == "" {
		return client.Config{}, errors.New("OUTPUT_FILE environment variable is required")
	}

	batchSize, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil || batchSize <= 0 {
		batchSize = 32
	}

	return client.Config{
		ServerHost: serverHost,
		ServerPort: serverPort,
		AgencyId:   agencyId,
		InputFile:  inputFile,
		OutputFile: outputFile,
		BatchSize:  batchSize,
	}, nil
}

func run() int {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)

	config, err := loadConfig()
	if err != nil {
		logger.Error("load-config", logger.Fail, "err", err)
		return 1
	}

	newClient, err := client.NewClient(config)
	if err != nil {
		logger.Error("client-new", logger.Fail, "err", err)
		return 1
	}

	go func() {
		<-stop
		err := newClient.Close()
		if err != nil {
			return
		}
	}()

	if err := newClient.Run(); err != nil {
		logger.Error("client-run", logger.Fail, "err", err)
	}
	return 0
}

func main() {
	os.Exit(run())
}
