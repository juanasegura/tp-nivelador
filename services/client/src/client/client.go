package client

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/logger"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/lottery/protocol"
	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const CONNECTION_ATTEMPTS_MAX = 3
const CONNECTION_ATTEMPS_DELAY_MS = 200

const (
	// Campos del CSV de entrada (first_name,last_name,document,birthdate,number)
	FieldFirstName = 0
	FieldLastName  = 1
	FieldDocument  = 2
	FieldBirthdate = 3
	FieldNumber    = 4
	ExpectedFields = 5

	// Auxiliares numéricos
	Base10 = 10
)

type ClientConfig struct {
	ServerHost string
	ServerPort string
	AgencyId   string
	InputFile  string
	OutputFile string
	BatchSize  int
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

	inputFile, err := os.Open(client.config.InputFile)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo %q: %w", client.config.InputFile, err)
	}
	defer func() {
		if err := inputFile.Close(); err != nil {
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

	scanner := bufio.NewScanner(inputFile)
	var batch []protocol.Bet
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		bet, err := parseLine(line, client.config.AgencyId)
		if err != nil {
			return fmt.Errorf("error al parsear línea %q: %w", line, err)
		}
		batch = append(batch, bet)
		if len(batch) >= client.config.BatchSize {
			if err := client.sendBets(batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error al leer archivo %w", err)
	}
	if len(batch) > 0 {
		if err := client.sendBets(batch); err != nil {
			return err
		}
	}

	if err := safe_socket.SendAll(client.conn, protocol.MakePacketNoMoreBets()); err != nil {
		return fmt.Errorf("error al enviar no-more-bets por socket %w", err)
	}

	packet, err := protocol.ReadMessage(client.conn)
	if err != nil {
		return fmt.Errorf("error al recibir winners por socket %w", err)
	}
	if !packet.IsBets() {
		return fmt.Errorf("se esperaba un paquete de winners, se recibió otro tipo")
	}
	winners, err := protocol.BetsFromBytes(packet.Payload())
	if err != nil {
		return fmt.Errorf("error al deserializar winners %w", err)
	}

	for _, w := range winners {
		betLine := fmt.Sprintf("%s,%s,%d,%s,%d\n",
			w.FirstName, w.LastName, w.Document, w.Birthdate, w.Number)
		if _, err := writer.WriteString(betLine); err != nil {
			return fmt.Errorf("error al escribir en archivo: %w", err)
		}
	}

	if err := safe_socket.SendAll(client.conn, protocol.MakePacketAck()); err != nil {
		return fmt.Errorf("error al enviar ack de winners por socket %w", err)
	}

	logger.Info(mainAction, logger.Success, "agency-id", client.config.AgencyId, "winners", len(winners))

	return nil
}

func parseLine(line, agencyId string) (protocol.Bet, error) {
	fields := strings.Split(line, ",")
	if len(fields) != ExpectedFields {
		return protocol.Bet{}, fmt.Errorf("se esperaban %d campos, hay %d", ExpectedFields, len(fields))
	}
	document, err := strconv.ParseUint(fields[FieldDocument], Base10, 32)
	if err != nil {
		return protocol.Bet{}, err
	}
	number, err := strconv.ParseUint(fields[FieldNumber], Base10, 32)
	if err != nil {
		return protocol.Bet{}, err
	}
	agency, err := strconv.ParseUint(agencyId, Base10, 32)
	if err != nil {
		return protocol.Bet{}, err
	}
	return protocol.Bet{
		AgencyId:  uint32(agency),
		FirstName: fields[FieldFirstName],
		LastName:  fields[FieldLastName],
		Document:  uint32(document),
		Birthdate: fields[FieldBirthdate],
		Number:    uint32(number),
	}, nil
}

func (client *Client) sendBets(bets []protocol.Bet) error {
	if err := safe_socket.SendAll(client.conn, protocol.MakePacketBets(bets)); err != nil {
		return fmt.Errorf("error al enviar bets por socket %w", err)
	}
	packet, err := protocol.ReadMessage(client.conn)
	if err != nil {
		return fmt.Errorf("error al recibir ack por socket %w", err)
	}
	if !packet.IsAck() {
		return fmt.Errorf("se esperaba un ack, se recibió otro tipo")
	}
	return nil
}
