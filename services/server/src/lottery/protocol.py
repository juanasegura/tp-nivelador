from lottery.bet import Bet

UINT32_SIZE = 4  # bytes de un uint32
BIRTHDATE_LEN = 10  # tamaño de fecha (YYYY-MM-DD)

TYPE_BET = 0
TYPE_ACK = 1
TYPE_NO_MORE_BETS = 2
TYPE_WINNERS = 3


def to_bytes(bet: Bet) -> bytes:
    buf = []
    buf += bet.agency_id.to_bytes(UINT32_SIZE, "big")

    first = bet.first_name.encode("utf-8")
    buf += len(first).to_bytes(UINT32_SIZE, "big") + first

    last = bet.last_name.encode("utf-8")
    buf += len(last).to_bytes(UINT32_SIZE, "big") + last

    buf += bet.document.to_bytes(UINT32_SIZE, "big")

    buf += bet.birthdate.encode("utf-8")

    buf += bet.number.to_bytes(UINT32_SIZE, "big")
    return buf

def from_bytes(data: bytes) -> Bet:
    pos = 0
    agency_id = int.from_bytes(data[pos: pos + UINT32_SIZE], "big")
    pos += UINT32_SIZE

    first_len = int.from_bytes(data[pos: pos + UINT32_SIZE], "big")
    pos += UINT32_SIZE
    first_name = data[pos: pos + first_len].decode("utf-8")
    pos += first_len

    last_len = int.from_bytes(data[pos: pos + UINT32_SIZE], "big")
    pos += UINT32_SIZE
    last_name = data[pos : pos + last_len].decode("utf-8")
    pos += last_len

    document = int.from_bytes(data[pos: pos + UINT32_SIZE], "big")
    pos += UINT32_SIZE

    birthdate = data[pos: pos + BIRTHDATE_LEN].decode("utf-8")
    pos += BIRTHDATE_LEN

    number = int.from_bytes(data[pos : pos + UINT32_SIZE], "big")
    return Bet(agency_id, first_name, last_name, document, birthdate, number)

def make_packet(msg_type: int, payload: bytes = b"") -> bytes:
    return (
        bytes([msg_type])
        + len(payload).to_bytes(UINT32_SIZE, "big")
        + payload
    )

def make_packet_no_more_bets() -> bytes:
    return make_packet(TYPE_NO_MORE_BETS)


def make_packet_ack() -> bytes:
    return make_packet(TYPE_ACK)


def make_packet_bet(bet: Bet) -> bytes:
    return make_packet(TYPE_BET, to_bytes(bet))

def read_message(sock):
    header = safe_socket.recv_all(sock, HEADER_SIZE)
    msg_type = header[0]
    length = int.from_bytes(header[1:HEADER_SIZE], "big")
    payload = safe_socket.recv_all(sock, length) if length else b""
    return msg_type, payload


def winners_to_bytes(winners: list[Bet]) -> bytes:
    buf = len(winners).to_bytes(UINT32_SIZE, "big")
    for w in winners:
        payload = to_bytes(w)
        buf += len(payload).to_bytes(UINT32_SIZE, "big") + payload
    return buf


def make_packet_winners(winners: list[Bet]) -> bytes:
    return write_packet(TYPE_WINNERS, winners_to_bytes(winners))
