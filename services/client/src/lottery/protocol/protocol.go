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
	HeaderSize   = 5  // 1 byte de tipo + 4 de len de payload

	TypeBets       = 0
	TypeAck        = 1
	TypeNoMoreBets = 2
)

type Bet struct {
	AgencyId  uint32
	FirstName string
	LastName  string
	Document  uint32
	Birthdate string
	Number    uint32
}

type Packet struct {
	typ     byte
	payload []byte
}

func (p *Packet) IsBets() bool       { return p.typ == TypeBets }
func (p *Packet) IsAck() bool        { return p.typ == TypeAck }
func (p *Packet) IsNoMoreBets() bool { return p.typ == TypeNoMoreBets }
func (p *Packet) Payload() []byte    { return p.payload }

func BetsToBytes(bets []Bet, agencyId uint32) []byte {
	buf := make([]byte, 0, Uint32Size*2)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(bets)))
	buf = binary.BigEndian.AppendUint32(buf, agencyId)
	for _, bet := range bets {
		buf = append(buf, BetToBytes(bet)...)
	}
	return buf
}

func BetToBytes(bet Bet) []byte {
	buf := make([]byte, 0, 60)

	buf = append(buf, byte(len(bet.FirstName)))
	buf = append(buf, bet.FirstName...)

	buf = append(buf, byte(len(bet.LastName)))
	buf = append(buf, bet.LastName...)

	buf = binary.BigEndian.AppendUint32(buf, bet.Document)

	buf = append(buf, bet.Birthdate...)

	buf = binary.BigEndian.AppendUint32(buf, bet.Number)

	return buf
}

func BetsFromBytes(payload []byte) ([]Bet, error) {
	if len(payload) < Uint32Size*2 {
		return nil, fmt.Errorf("payload de bets incompleto")
	}
	count := int(binary.BigEndian.Uint32(payload[:Uint32Size]))
	agencyId := binary.BigEndian.Uint32(payload[Uint32Size : Uint32Size*2])
	pos := Uint32Size * 2
	bets := make([]Bet, 0, count)
	for range count {
		bet, newPos, err := BetFromBytes(payload, pos, agencyId)
		if err != nil {
			return nil, err
		}
		pos = newPos
		bets = append(bets, bet)
	}
	return bets, nil
}

func BetFromBytes(b []byte, pos int, agencyId uint32) (Bet, int, error) {
	firstName, pos, err := readString(b, pos)
	if err != nil {
		return Bet{}, 0, err
	}

	lastName, pos, err := readString(b, pos)
	if err != nil {
		return Bet{}, 0, err
	}

	if pos+Uint32Size > len(b) {
		return Bet{}, 0, fmt.Errorf("payload incompleto")
	}
	document := binary.BigEndian.Uint32(b[pos : pos+Uint32Size])
	pos += Uint32Size

	if pos+BirthdateLen > len(b) {
		return Bet{}, 0, fmt.Errorf("payload incompleto")
	}
	birthday := string(b[pos : pos+BirthdateLen])
	pos += BirthdateLen

	if pos+Uint32Size > len(b) {
		return Bet{}, 0, fmt.Errorf("payload incompleto")
	}
	number := binary.BigEndian.Uint32(b[pos : pos+Uint32Size])
	pos += Uint32Size

	return Bet{
		AgencyId:  agencyId,
		FirstName: firstName,
		LastName:  lastName,
		Document:  document,
		Birthdate: birthday,
		Number:    number,
	}, pos, nil
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

func makePacket(pktType byte, payload []byte) []byte {
	packet := make([]byte, HeaderSize, HeaderSize+len(payload))
	packet[0] = pktType
	binary.BigEndian.PutUint32(packet[1:5], uint32(len(payload)))
	return append(packet, payload...)
}

func MakePacketNoMoreBets() []byte {
	return makePacket(TypeNoMoreBets, nil)
}

func MakePacketAck() []byte {
	return makePacket(TypeAck, nil)
}

func MakePacketBets(bets []Bet) []byte {
	return makePacket(TypeBets, BetsToBytes(bets, bets[0].AgencyId))
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
