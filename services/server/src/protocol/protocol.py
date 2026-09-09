import safe_socket
from lottery.bet import Bet

UINT32_SIZE = 4  # bytes de un uint32
UINT8_SIZE = 1  # bytes de un uint8
BIRTHDATE_LEN = 10  # tamaño de fecha (YYYY-MM-DD)
HEADER_SIZE = 5  # 1 byte de type + 4 de len de payload

TYPE_BETS = 0
TYPE_ACK = 1
TYPE_NO_MORE_BETS = 2


# definición del tipo paquete
class Packet:
    def __init__(self, msg_type: int, payload: bytes):
        self._type = msg_type
        self._payload = payload

    def is_bets(self) -> bool:
        return self._type == TYPE_BETS

    def is_ack(self) -> bool:
        return self._type == TYPE_ACK

    def is_no_more_bets(self) -> bool:
        return self._type == TYPE_NO_MORE_BETS

    def payload(self) -> bytes:
        return self._payload


# serialización
def bets_to_bytes(bets: list[Bet], agency_id: int) -> bytes:
    buf = len(bets).to_bytes(UINT32_SIZE, "big")
    buf += agency_id.to_bytes(UINT32_SIZE, "big")
    for bet in bets:
        buf += bet_to_bytes(bet)
    return buf

def bet_to_bytes(bet: Bet) -> bytes:
    buf = b""

    first = bet.first_name.encode("utf-8")
    buf += len(first).to_bytes(UINT8_SIZE, "big") + first

    last = bet.last_name.encode("utf-8")
    buf += len(last).to_bytes(UINT8_SIZE, "big") + last

    buf += bet.document.to_bytes(UINT32_SIZE, "big")

    buf += bet.birthdate.encode("utf-8")

    buf += bet.number.to_bytes(UINT32_SIZE, "big")
    return buf


def bets_from_bytes(data: bytes) -> list[Bet]:
    count = int.from_bytes(data[:UINT32_SIZE], "big")
    agency_id = int.from_bytes(data[UINT32_SIZE : UINT32_SIZE * 2], "big")
    bets = []
    pos = UINT32_SIZE * 2
    for _ in range(count):
        bet, pos = bet_from_bytes(data, pos, agency_id)
        bets.append(bet)
    return bets


def bet_from_bytes(data: bytes, pos: int, agency_id: int) -> tuple[Bet, int]:
    first_len = int(data[pos])
    pos += UINT8_SIZE
    first_name = data[pos : pos + first_len].decode("utf-8")
    pos += first_len

    last_len = int(data[pos])
    pos += UINT8_SIZE
    last_name = data[pos : pos + last_len].decode("utf-8")
    pos += last_len

    document = int.from_bytes(data[pos : pos + UINT32_SIZE], "big")
    pos += UINT32_SIZE

    birthdate = data[pos : pos + BIRTHDATE_LEN].decode("utf-8")
    pos += BIRTHDATE_LEN

    number = int.from_bytes(data[pos : pos + UINT32_SIZE], "big")
    pos += UINT32_SIZE

    return Bet(agency_id, first_name, last_name, document, birthdate, number), pos


# creadores de paquetes
def make_packet(msg_type: int, payload: bytes = b"") -> bytes:
    return bytes([msg_type]) + len(payload).to_bytes(UINT32_SIZE, "big") + payload


def make_packet_no_more_bets() -> bytes:
    return make_packet(TYPE_NO_MORE_BETS)


def make_packet_ack() -> bytes:
    return make_packet(TYPE_ACK)


def make_packet_bets(bets: list[Bet], agency_id: int) -> bytes:
    return make_packet(TYPE_BETS, bets_to_bytes(bets, agency_id))


# leer del socket para formar un paquete
def read_message(sock) -> Packet:
    header = safe_socket.recv_all(sock, HEADER_SIZE)
    msg_type = header[0]
    length = int.from_bytes(header[1:HEADER_SIZE], "big")
    payload = safe_socket.recv_all(sock, length) if length else b""
    return Packet(msg_type, payload)
