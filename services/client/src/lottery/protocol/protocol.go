package protocol

import (
	"encoding/binary"
	"fmt"
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

func MakePacket(pktType byte, payload []byte) []byte {
	packet := make([]byte, HeaderSize, HeaderSize+len(payload))
	packet[0] = pktType
	binary.BigEndian.PutUint32(packet[1:5], uint32(len(payload)))
	return append(packet, payload...)
}

func CreateAck() []byte {
	return MakePacket(TypeAck, nil)
}

func CreateNoMoreBets() []byte {
	return MakePacket(TypeNoMoreBets, nil)
}

func CreateBet(bet Bet) []byte {
	return MakePacket(TypeBet, ToBytes(bet))
}
