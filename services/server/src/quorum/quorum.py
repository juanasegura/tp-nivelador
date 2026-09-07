import multiprocessing

def quorum_process(
    requests_q: multiprocessing.Queue,
    ready_q: multiprocessing.Queue,
    agency_quorum_min: int,
):
    received = 0
    while True:
        requests_q.get()
        received += 1
        if received == agency_quorum_min:
            for _ in range(agency_quorum_min):
                ready_q.put(True)
        elif received > agency_quorum_min:
            ready_q.put(True)