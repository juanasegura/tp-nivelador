import fcntl
import os
from collections.abc import Iterator

from lottery import protocol
from lottery.bet import Bet

UINT32_SIZE = 4

LOCK_SH = fcntl.LOCK_SH
LOCK_EX = fcntl.LOCK_EX
LOCK_UN = fcntl.LOCK_UN


def _write_all(fd: int, data: bytes):
    total = 0
    while total < len(data):
        written = os.write(fd, data[total:])
        if written == 0:
            raise ValueError("error al escribir")
        total += written


def _batch_to_bytes(bets: list[Bet], agency_id: int) -> bytes:
    payload = protocol.bets_to_bytes(bets, agency_id)
    return len(payload).to_bytes(UINT32_SIZE, "big") + payload


def store_bets(path: str, bets: list[Bet], agency_id: int) -> None:
    with open(path, "ab") as file:
        fcntl.flock(file.fileno(), LOCK_EX)
        try:
            _write_all(file.fileno(), _batch_to_bytes(bets, agency_id))
        finally:
            fcntl.flock(file.fileno(), LOCK_UN)

# LORD: el iterator
def iter_bets(path: str) -> Iterator[Bet]:
    try:
        file = open(path, "rb")
    except FileNotFoundError:
        return

    with file:
        fcntl.flock(file.fileno(), LOCK_SH)
        try:
            while True:
                header = file.read(UINT32_SIZE)
                if not header:
                    break
                if len(header) != UINT32_SIZE:
                    raise ValueError("header incompleto")
                length = int.from_bytes(header, "big")
                payload = file.read(length)
                if len(payload) != length:
                    raise ValueError("lectura incompleta")
                for bet in protocol.bets_from_bytes(payload):
                    yield bet
        finally:
            fcntl.flock(file.fileno(), LOCK_UN)