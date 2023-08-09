from dataclasses import dataclass
import gzip
import sys
import traceback
from urllib.parse import urlparse
import zlib
import requests
import threading

from collections import namedtuple
from typing import Callable, List, Optional
from time import time, sleep
from queue import Queue

from runner import elastic

# TODO: making this a class

url = None
hostname = None
port = None
method = None
body_type = None
reqs_limit = 10
_reqs_current = 0
_reqs_checkpoint_s = 10
_reqs_checkpoint = _reqs_checkpoint_s * 20
_stop_ev = threading.Event()
_ready_ev = threading.Event()

last_response: requests.Response = None
last_err: requests.Response = None
last_exc = None

@dataclass
class WorkerResult:
    start_t: float
    end_t: float
    resp_status_code: int
    resp_headers: dict
    resp_content_encoding: str
    resp_content_length: int
    resp_server_name: str
    resp_edge_proxy: bool
    req_url: str
    req_port: Optional[int]
    req_scheme: str
    req_headers: dict
    wk_options: dict


results: Queue = None

def setup(**kwargs):
    global url, method, hostname, body_type, port, reqs_limit, results

    url = kwargs.get('url', None)
    method = kwargs.get('method', None)
    hostname = kwargs.get('hostname', None)
    port = kwargs.get('port', None)
    body_type = kwargs.get('body_type', None)
    reqs_limit = kwargs.get('reqs_limit', None)

    results = elastic.get_queue()
    

def reset():
    global _stop_ev, _ready_ev
    global _reqs_current, _reqs_checkpoint, _reqs_checkpoint_s
    global last_response, last_err, last_exc

    _stop_ev = threading.Event()
    _ready_ev = threading.Event()

    _reqs_current = 0
    _reqs_checkpoint_s = 10
    _reqs_checkpoint = _reqs_checkpoint_s * 20

    last_response = None
    last_err = None
    last_exc = None

@dataclass
class FailedRequest:
    status_code: int
    headers: dict
    request: requests.Request



def request_worker(uuid: str, options: dict, next_delay: Callable, next_body: Callable, session: requests.Session):
    global _reqs_current, _reqs_checkpoint, _reqs_checkpoint_s
    global last_response, last_err, last_exc

    gzip_on = options.get('gzip')

    try:
        headers = {'Host': hostname}

        if body_type is not None:
            headers['Content-Type'] = body_type
        
        if gzip_on:
            headers['Accept-Encoding'] = 'gzip'
            headers['Content-Encoding'] = 'gzip'

        while not _stop_ev.is_set() and _reqs_current < reqs_limit:
            _reqs_current += 1
            req_no = _reqs_current

            body = next_body()

            if gzip_on:
                if isinstance(body, str):
                    body = gzip.compress(body.encode())
                elif isinstance(body, bytes):
                    body = gzip.compress(body)
            
            # if reqs_current % reqs_checkpoint == 0:
            #   print(f'Completed {reqs_current} requests')

            start_t = time()
            try:
                req = requests.Request(method=method, url=url, headers=headers, data=body)
                req = session.prepare_request(req)
                last_response = resp = session.send(req)
                content_encoding = resp.encoding
                content_length = len(resp.content)
                status_code = resp.status_code
                name = resp.headers.get('x-k-node-name', 'unknown')
                edge_proxy = resp.headers.get('x-knative-edge-proxy', 'false').lower() == 'true'

                if not resp.ok:
                    last_err = resp
                    # stop_ev.set()
            except Exception:
                last_exc = sys.exc_info()
                content_length = 0
                content_encoding = ''
                status_code = -1
                name = 'none/fail'
                edge_proxy=False

                resp = FailedRequest(status_code, {}, req)

            if req_no % _reqs_checkpoint_s == 0 and not _stop_ev.is_set():
                print('.', end='', flush=True)

                if _reqs_current >= _reqs_checkpoint:
                    _reqs_checkpoint *= 10
                    _reqs_checkpoint_s *= 10

            if _stop_ev.is_set():
                break

            # if not stop_ev:
            #   results.append((time() - start_t, req.status_code))
            
            _url = urlparse(resp.request.url)

            results.put(
                uuid,
                WorkerResult(
                    start_t=start_t,
                    end_t=time(),
                    resp_status_code=status_code,
                    resp_headers=dict(resp.headers),
                    resp_content_encoding=content_encoding,
                    resp_content_length=content_length,
                    resp_server_name=name,
                    resp_edge_proxy=edge_proxy,
                    req_url=resp.request.url,
                    req_port=_url.port,
                    req_scheme=str(_url.scheme),
                    req_headers=dict(resp.request.headers),
                    wk_options=options,
                )
            )

            if req_no >= reqs_limit:
                _stop_ev.set()

            if next_delay is None:
                break

            delay = next_delay()
            if delay is not None:
                _stop_ev.wait(delay)

        _ready_ev.set()
    except:
        traceback.print_exc()
    

def timeout_worker(limit):
    global _stop_ev

    if limit is None:
        return

    sleep(limit)
    _stop_ev.set()

