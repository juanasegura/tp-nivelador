import socket

def recv_all(sock: socket.socket, size: int) -> bytes:
    buf = bytearray()
    while len(buf) < size:
        chunk = sock.recv(size - len(buf))
        if not chunk:
            raise ConnectionError("conexion cerrada")
        buf += chunk
    return bytes(buf)

def send_all(sock: socket.socket, data: bytes):
    total = 0
    while total < len(data):
        sent = sock.send(data[total:])
        if sent == 0:
            raise ConnectionError("error al enviar")
        total += sent