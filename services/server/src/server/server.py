import multiprocessing
import signal
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
        self.server_socket = None
        self._terminating = False

    def run(self):
        action = "accept-connection"
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
            self.server_socket = server_socket
            signal.signal(signal.SIGTERM, self._shutdown)

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
            while not self._terminating:
                try:
                    logger.info(action, logger.LogResult.in_progress)
                    client_socket, _ = server_socket.accept()
                except Exception as e:
                    if self._terminating:
                        break
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
                handlers = self._check_live_handlers(handlers)

            for _ in handlers:
                ready_q.put(quorum.DIE_TOKEN)
            requests_q.put(quorum.DIE_TOKEN)

            for proc in handlers:
                proc.terminate()

            for proc in handlers:
                proc.join()
            quorum_proc.join()

            requests_q.close()
            ready_q.close()

    def _check_live_handlers(self, handlers):
        alive = []
        for proc in handlers:
            if proc.is_alive():
                alive.append(proc)
            else:
                proc.join()
        return alive

    def _shutdown(self, signum, frame):
        self._terminating = True
        if self.server_socket is not None:
            self.server_socket.close()