import multiprocessing
import socket
import client_handler
import logger
import quorum
from lottery.lottery import Lottery


class Server:
    def __init__(self, server_host: str, server_port: int, storage_path: str, agency_quorum_min: int):
        self.server_host = server_host
        self.server_port = server_port
        self.storage_path = storage_path
        self.agency_quorum_min = agency_quorum_min
        self.lottery = Lottery(storage_path)


    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            server_socket.bind((self.server_host, self.server_port))
            server_socket.listen()

            requests_q = multiprocessing.Queue()
            ready_q = multiprocessing.Queue()
            quorum_proc = multiprocessing.Process(
                target=quorum.quorum_process,
                args=(requests_q, ready_q, self.agency_quorum_min),
                daemon=True,
            )
            quorum_proc.start()

            handlers = []
            while True:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    logger.error(action, logger.LogResult.fail)
                    raise e
                logger.info(action, logger.LogResult.success)

                proc = multiprocessing.Process(
                    target=client_handler.handle_client,
                    args=(client_socket, self.storage_path, requests_q, ready_q),
                    daemon=True,
                )
                proc.start()
                client_socket.close()
                handlers.append(proc)
                handlers = _join_procs(handlers)

def _join_procs(handlers):
    alive = []
    for proc in handlers:
        if proc.is_alive():
            alive.append(proc)
        else:
            proc.join()
    return alive