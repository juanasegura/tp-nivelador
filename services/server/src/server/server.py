import multiprocessing
import signal
import socket

from lottery.lottery import Lottery

import client_handler
import logger
import quorum


class Server:
    def __init__(
        self,
        server_host: str,
        server_port: int,
        storage_path: str,
        agency_quorum_min: int,
    ):
        self._server_host = server_host
        self._server_port = server_port
        self._lottery = Lottery(storage_path)
        self._agency_quorum_min = agency_quorum_min
        self._server_socket = None
        self._requests_q = None
        self._ready_q = None
        self._quorum_proc = None
        self._handlers = []
        self._terminating = False

    def run(self):
        action = "accept-connection"
        try:
            self._start_resources()

            while not self._terminating:
                logger.info(action, logger.LogResult.in_progress)
                client_socket, _ = self._server_socket.accept()
                logger.info(action, logger.LogResult.success)

                proc = multiprocessing.Process(
                    target=client_handler.handle_client,
                    args=(
                        client_socket,
                        self._lottery,
                        self._requests_q,
                        self._ready_q,
                    ),
                    daemon=True,
                )
                proc.start()
                client_socket.close()
                self._handlers.append(proc)
                self._check_live_handlers()
        except Exception:
            if not self._terminating:
                logger.error(action, logger.LogResult.fail)
                raise
        finally:
            if not self._terminating:
                self._stop_resources()

    def _start_resources(self):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        signal.signal(signal.SIGTERM, self._shutdown)

        self._server_socket.bind((self._server_host, self._server_port))
        self._server_socket.listen()

        self._requests_q = multiprocessing.Queue()
        self._ready_q = multiprocessing.Queue()
        self._quorum_proc = multiprocessing.Process(
            target=quorum.quorum_process,
            args=(self._requests_q, self._ready_q, self._agency_quorum_min),
            daemon=True,
        )
        self._quorum_proc.start()
        self._handlers = []

    def _stop_resources(self):
        if self._server_socket is not None:
            self._server_socket.close()

        for _ in self._handlers:
            self._ready_q.put(quorum.DIE_TOKEN)
        self._requests_q.put(quorum.DIE_TOKEN)

        for proc in self._handlers:
            proc.terminate()

        for proc in self._handlers:
            proc.join()
        if self._quorum_proc is not None:
            self._quorum_proc.join()
        if self._requests_q is not None:
            self._requests_q.close()
            self._requests_q.join_thread()
        if self._ready_q is not None:
            self._ready_q.close()
            self._ready_q.join_thread()

    def _check_live_handlers(self):
        alive = []
        for proc in self._handlers:
            if proc.is_alive():
                alive.append(proc)
            else:
                proc.join()
        self._handlers = alive

    def _shutdown(self, signum, frame):
        self._terminating = True
        self._stop_resources()
