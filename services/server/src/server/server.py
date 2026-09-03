import socket
import logger
import safe_socket
import protocol
from lottery.lottery import Lottery

class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str):
        self.server_host = server_host
        self.server_port = server_port
        self.lottery = Lottery(storage_path)

    def _handle_client(self, client_socket):
        action = "handle-client"
        try:
            logger.info(action, logger.LogResult.in_progress)
            while True:
                packet = protocol.read_message(client_socket)
                if packet.is_bet():
                    bet = protocol.bet_from_bytes(packet.payload())
                    self.lottery.store_bets([bet])
                    safe_socket.send_all(
                        client_socket,
                        protocol.make_packet_ack(),
                    )
                elif packet.is_no_more_bets():
                    winners = [
                        b for b in self.lottery.load_bets() if self.lottery.has_won(b)
                    ]
                    safe_socket.send_all(
                        client_socket,
                        protocol.make_packet_winners(winners),
                    )
                    break
        except Exception as e:
            logger.error(
                action, logger.LogResult.fail
            )
            raise e
        finally:
            client_socket.close()

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                self._handle_client(client_socket)
