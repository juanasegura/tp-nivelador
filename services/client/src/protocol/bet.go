package protocol

import (
	"encoding/binary"
	"fmt"
)

const (
	Uint32Size   = 4  // bytes de un uint32
	Uint8Size    = 1  // bytes de un uint8
	BirthdateLen = 10 // tamaño de fecha (YYYY-MM-DD)
)

type Bet struct {
	AgencyId  uint32
	FirstName string
	LastName  string
	Document  uint32
	Birthdate string
	Number    uint32
}

func BetsToBytes(bets []Bet, agencyId uint32) []byte {
	buf := make([]byte, 0, Uint32Size*2)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(bets)))
	buf = binary.BigEndian.AppendUint32(buf, agencyId)
	for _, bet := range bets {
		buf = append(buf, betToBytes(bet)...)
	}
	return buf
}

func betToBytes(bet Bet) []byte {
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
		bet, newPos, err := betFromBytes(payload, pos, agencyId)
		if err != nil {
			return nil, err
		}
		pos = newPos
		bets = append(bets, bet)
	}
	return bets, nil
}

func betFromBytes(b []byte, pos int, agencyId uint32) (Bet, int, error) {
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
