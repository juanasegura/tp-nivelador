import socket

import logger
import safe_socket
import storage
from lottery import protocol
from lottery.lottery import Lottery


def handle_client(
    client_socket: socket.socket,
    storage_path: str,
    requests_q,
    ready_q,
):
    action = "handle-client"
    lottery = Lottery(storage_path)
    client_agency = None
    try:
        logger.info(action, logger.LogResult.in_progress)
        while True:
            packet = protocol.read_message(client_socket)
            if packet.is_bets():
                bets = protocol.bets_from_bytes(packet.payload())
                client_agency = bets[0].agency_id
                storage.store_bets(storage_path, bets, client_agency)
                safe_socket.send_all(client_socket, protocol.make_packet_ack())
            elif packet.is_no_more_bets():
                requests_q.put(client_agency)
                ready_q.get()
                winners = [
                    b
                    for b in storage.iter_bets(storage_path)
                    if lottery.has_won(b) and b.agency_id == client_agency
                ]
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