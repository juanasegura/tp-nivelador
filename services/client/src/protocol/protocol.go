package protocol

import (
	"encoding/binary"
	"net"

	"github.com/7574-sistemas-distribuidos/tp-nivelador/src/safe_socket"
)

const HeaderSize = 5 // 1 byte de tipo + 4 de len de payload

const (
	TypeBets       = 0
	TypeAck        = 1
	TypeNoMoreBets = 2
)

type Packet struct {
	typ     byte
	payload []byte
}

func (p *Packet) IsBets() bool       { return p.typ == TypeBets }
func (p *Packet) IsAck() bool        { return p.typ == TypeAck }
func (p *Packet) IsNoMoreBets() bool { return p.typ == TypeNoMoreBets }
func (p *Packet) Payload() []byte    { return p.payload }

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
