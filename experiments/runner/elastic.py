from datetime import datetime
import multiprocessing as mp
from random import random
from time import sleep
import uuid
import elasticsearch
import pytz

_q: 'QueueProxy' = None
_p: mp.Process = None

def start(args):
    global _q, _p

    esclient = elasticsearch.Elasticsearch(args.es_host, basic_auth=(args.es_user, args.es_pass))

    if not esclient.ping():
        raise Exception(f'Could not connect to elastic: {args.es_host}')

    q = QueueProxy()

    _p = mp.Process(target=elastic_reporter, args=[q.raw_queue(), esclient], daemon=True)
    _p.start()

    _q = q

def stop():
    _q.put('STOP')
    _p.join()

def get_queue():
    return _q

def flush():
    _q.put('FLUSH')

def elastic_reporter(q: mp.Queue, esclient: elasticsearch.Elasticsearch):
    import threading

    stop_cond = False

    def periodic_flush():
        nonlocal stop_cond

        while not stop_cond:
            sleep(10)
            batch.flush()

    batch = ElasticBatch(esclient, 'experiments')

    try:
        while True:
            try:
                msg = q.get(True)
            except KeyboardInterrupt:
                batch.flush()
                return

            if msg[0] == 'STOP':
                batch.flush()
                return
            elif msg[0] == 'FLUSH':
                batch.flush()
            else:
                uuid, result = msg

                batch.add({
                    '@timestamp': datetime.fromtimestamp(result.start_t, pytz.UTC),
                    'experiment': {
                        'id': uuid,
                        'type': 'request',
                        'worker': result.wk_options,
                    },
                    'server': {
                        'name': result.resp_server_name,
                        'proxied': result.resp_edge_proxy,
                    },
                    'response': {
                        'status_code': result.resp_status_code,
                        'duration': (result.end_t - result.start_t) * 1e6,
                        'content_length': result.resp_content_length,
                        'headers': result.resp_headers,
                    },
                    'request': {
                        'url': result.req_url,
                        'port': result.req_port,
                        'scheme': result.req_scheme,
                        'headers': result.req_headers
                    },
                })
    finally:
        stop_cond = True

class QueueProxy:
    def __init__(self, *args, **kwargs) -> None:
        self._data = []
        self._queue = mp.Queue(*args, **kwargs)

    def raw_queue(self):
        return self._queue

    def put(self, *item):
        self._queue.put(item)

        try:
            self._data.append(item[1])
        except Exception:
            pass

    def __iter__(self):
        return self._data.__iter__()
    
    def __len__(self):
        return len(self._data)


class ElasticBatch:
    def __init__(self, esclient: elasticsearch.Elasticsearch, index: str):
        self.esclient = esclient
        self.index = index
        self._batch = []
        self._max_size = 32

    def flush(self):
        if len(self._batch) > 0:
            self._push()

        # pass

    def _push(self):
        exc = None

        for _ in range(3):
            try:
                self.esclient.bulk(index=self.index, body=self._batch)
                self._batch = []
                exc = None
                break
            except Exception as e:
                exc = e
                sleep(random())
        
        if exc is not None:
            raise exc

    def add(self, doc: dict):
        self._batch.append({
            'index': {
                '_id': str(uuid.uuid4()),
                '_index': self.index,
            }
        })

        self._batch.append(doc)

        if len(self._batch) >= self._max_size:
            self._push()

        # self.esclient.index(index=self.index, document=doc)
