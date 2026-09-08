import fcntl
import signal
import socket
import logger
import safe_socket
from lottery import protocol
from lottery.lottery import Lottery
from quorum import DIE_TOKEN

def handle_client(
    client_socket: socket.socket,
    storage_path: str,
    requests_q,
    ready_q,
):
    action = "handle-client"
    lottery = Lottery(storage_path)
    client_agency = None

    # defino como closure el manejo de la señal de fin
    def _terminate_handler(signum, frame):
        client_socket.close()
        requests_q.close()
        ready_q.close()

    signal.signal(signal.SIGTERM, _terminate_handler)

    try:
        logger.info(action, logger.LogResult.in_progress)
        while True:
            packet = protocol.read_message(client_socket)
            if packet.is_bets():
                bets = protocol.bets_from_bytes(packet.payload())
                client_agency = bets[0].agency_id
                _store_bets_with_lock(storage_path, bets)
                safe_socket.send_all(client_socket, protocol.make_packet_ack())
            elif packet.is_no_more_bets():
                requests_q.put(client_agency)
                token = ready_q.get()
                if token == DIE_TOKEN:
                    break
                winners = _load_bets_with_lock(storage_path, lottery, client_agency)
                safe_socket.send_all(
                    client_socket,
                    protocol.make_packet_bets(winners, client_agency),
                )
                ack = protocol.read_message(client_socket)
                if not ack.is_ack():
                    raise ValueError("se esperaba un ack del cliente")
                break
    except Exception as e:
        logger.error(action, logger.LogResult.fail)
        raise e
    finally:
        client_socket.close()
        requests_q.close()
        ready_q.close()


def _store_bets_with_lock(storage_path: str, bets) -> None:
    storage_file = open(storage_path, "a+")
    try:
        fcntl.flock(storage_file.fileno(), fcntl.LOCK_EX)
        Lottery(storage_path).store_bets(bets)
    finally:
        fcntl.flock(storage_file.fileno(), fcntl.LOCK_UN)
        storage_file.close()


def _load_bets_with_lock(storage_path: str, lottery: Lottery, client_agency: int):
    storage_file = open(storage_path, "a+")
    try:
        fcntl.flock(storage_file.fileno(), fcntl.LOCK_SH)
        return [
            b
            for b in lottery.load_bets()
            if lottery.has_won(b) and b.agency_id == client_agency
        ]
    finally:
        fcntl.flock(storage_file.fileno(), fcntl.LOCK_UN)
        storage_file.close()