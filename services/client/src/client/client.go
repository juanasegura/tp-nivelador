package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
}

type Client struct {
	conn   net.Conn
	config ClientConfig
}

func NewClient(config ClientConfig) (*Client, error) {
	conn, err := connectToServer(config.ServerHost, config.ServerPort)
	if err != nil {
		logger.Warn("connect-to-server", logger.Fail)
		return nil, err
	}

	client := &Client{conn: conn, config: config}
	return client, nil
}

func connectToServer(host, port string) (net.Conn, error) {
	const action = "connect-to-server"
	var err error
	var conn net.Conn

	logger.Info(action, logger.InProgress)
	for i := range CONNECTION_ATTEMPTS_MAX {
		conn, err = net.Dial("tcp", host+":"+port)
		if err != nil {
			logger.Warn(action, logger.Fail, "attempt", i)
			time.Sleep(CONNECTION_ATTEMPS_DELAY_MS * time.Millisecond)
			continue
		}

		logger.Info(action, logger.Success)
		break
	}

	return conn, err
}

func (client *Client) Run() error {
	const mainAction = "send-bets"
	defer func() {
		if err := client.conn.Close(); err != nil {
			logger.Error("close-connection", logger.Fail, "err", err)
		}
	}()

	input_file, err := os.Open(client.config.InputFile)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo %q: %w", client.config.InputFile, err)
	}
	defer func() {
		if err := input_file.Close(); err != nil {
			logger.Error("close-input", logger.Fail, "err", err)
		}
	}()

	outputFile, err := os.Create(client.config.OutputFile)
	if err != nil {
		return fmt.Errorf("error al crear archivo %q: %w", client.config.OutputFile, err)
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			logger.Error("close-output", logger.Fail, "err", err)
		}
	}()
	writer := bufio.NewWriter(outputFile)
	defer func() {
		if err := writer.Flush(); err != nil {
			logger.Error("flush-output", logger.Fail, "err", err)
		}
	}()

	scanner := bufio.NewScanner(input_file)
	for scanner.Scan() {
		line := scanner.Text()
		bytes := []byte(line)
		err := safe_socket.SendAll(client.conn, bytes)
		if err != nil {
			return fmt.Errorf("error al enviar por socket %w", err)
		}
		response, err := safe_socket.RecvAll(client.conn, 1024)
		if err != nil {
			return fmt.Errorf("error al recibir por socket %w", err)
		}
		if _, err := writer.Write(response); err != nil {
			return fmt.Errorf("error al escribir en archivo: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error al leer archivo %w", err)
	}
	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId)

	return nil
}
