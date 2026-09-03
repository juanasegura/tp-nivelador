package protocol

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const (
	Uint32Size   = 4  // bytes de un uint32
	Uint8Size    = 1  // bytes de un uint8
	BirthdateLen = 10 // tamaño de fecha (YYYY-MM-DD)
	HeaderSize   = 5  // 1 byte de packet type, 4 de len de payload

	TypeBet        = 0
	TypeAck        = 1
	TypeNoMoreBets = 2
	TypeWinners    = 3
)

type Bet struct {
	AgencyId  uint32
	FirstName string
	LastName  string
	Document  uint32
	Birthdate string
	Number    uint32
}

func ToBytes(b Bet) []byte {
	buf := make([]byte, 0, 64)

	buf = binary.BigEndian.AppendUint32(buf, b.AgencyId)

	buf = append(buf, byte(len(b.FirstName)))
	buf = append(buf, b.FirstName...)

	buf = append(buf, byte(len(b.LastName)))
	buf = append(buf, b.LastName...)

	buf = binary.BigEndian.AppendUint32(buf, b.Document)

	buf = append(buf, b.Birthdate...)

	buf = binary.BigEndian.AppendUint32(buf, b.Number)

	return buf
}

func FromBytes(b []byte) (Bet, error) {
	if len(b) < Uint32Size {
		return Bet{}, fmt.Errorf("payload demasiado corto")
	}

	pos := 0
	agencyId := binary.BigEndian.Uint32(b[pos : pos+4])
	pos += Uint32Size

	firstName, pos, err := readString(b, pos)
	if err != nil {
		return Bet{}, err
	}

	lastName, pos, err := readString(b, pos)
	if err != nil {
		return Bet{}, err
	}

	if pos+Uint32Size > len(b) {
		return Bet{}, fmt.Errorf("payload incompleto")
	}
	document := binary.BigEndian.Uint32(b[pos : pos+4])
	pos += Uint32Size

	if pos+BirthdateLen > len(b) {
		return Bet{}, fmt.Errorf("payload incompleto")
	}
	birthday := string(b[pos : pos+10])
	pos += 10

	if pos+Uint32Size > len(b) {
		return Bet{}, fmt.Errorf("payload incompleto")
	}
	number := binary.BigEndian.Uint32(b[pos : pos+Uint32Size])

	return Bet{
		AgencyId:  agencyId,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: birthday,
		Number:    number,
	}, nil
}

func readString(b []byte, pos int) (string, int, error) {
	if pos+Uint8Size > len(b) {
		return "", 0, fmt.Errorf("payload incompleto")
	}
	strLen := int(b[pos])
	pos += Uint8Size
	if pos+strLen > len(b) {
		return "", 0, fmt.Errorf("payload incompleto")
	}
	str := string(b[pos : pos+strLen])
	pos += strLen
	return str, pos, nil
}

type Packet struct {
	typ     byte
	payload []byte
}

func (p *Packet) IsBet() bool        { return p.typ == TypeBet }
func (p *Packet) IsAck() bool        { return p.typ == TypeAck }
func (p *Packet) IsNoMoreBets() bool { return p.typ == TypeNoMoreBets }
func (p *Packet) IsWinners() bool    { return p.typ == TypeWinners }
func (p *Packet) Payload() []byte    { return p.payload }

func makePacket(pktType byte, payload []byte) []byte {
	packet := make([]byte, HeaderSize, HeaderSize+len(payload))
	packet[0] = pktType
	binary.BigEndian.PutUint32(packet[1:5], uint32(len(payload)))
	return append(packet, payload...)
}

func CreateAck() []byte {
	return makePacket(TypeAck, nil)
}

func CreateNoMoreBets() []byte {
	return makePacket(TypeNoMoreBets, nil)
}

func CreateBet(bet Bet) []byte {
	return makePacket(TypeBet, ToBytes(bet))
}

func ReadMessage(conn net.Conn) (*Packet, error) {
	header, err := safe_socket.RecvAll(conn, HeaderSize)
	if err != nil {
		return nil, err
	}
	msgType := header[0]
	length := binary.BigEndian.Uint32(header[1:5])
	payload, err := safe_socket.RecvAll(conn, int(length))
	if err != nil {
		return nil, err
	}
	return &Packet{typ: msgType, payload: payload}, nil
}

func FromWinners(payload []byte) ([]Bet, error) {
	if len(payload) < Uint32Size {
		return nil, fmt.Errorf("payload de winners incompleto")
	}
	count := int(binary.BigEndian.Uint32(payload[:Uint32Size]))
	pos := Uint32Size
	winners := make([]Bet, 0, count)
	for range count {
		if pos+Uint32Size > len(payload) {
			return nil, fmt.Errorf("winners incompleto")
		}
		betLen := int(binary.BigEndian.Uint32(payload[pos : pos+Uint32Size]))
		pos += Uint32Size
		if pos+betLen > len(payload) {
			return nil, fmt.Errorf("winners incompleto")
		}
		bet, err := FromBytes(payload[pos : pos+betLen])
		if err != nil {
			return nil, err
		}
		pos += betLen
		winners = append(winners, bet)
	}
	return winners, nil
}
