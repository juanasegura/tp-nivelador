import multiprocessing

def quorum_process(
    requests_q: multiprocessing.Queue,
    ready_q: multiprocessing.Queue,
    agency_quorum_min: int,
):
    agencies = set()
    while True:
        client_agency = requests_q.get()
        agencies.add(client_agency)
        if len(agencies) == agency_quorum_min:
            for _ in range(agency_quorum_min):
                ready_q.put(True)
        elif len(agencies) > agency_quorum_min:
            ready_q.put(True)