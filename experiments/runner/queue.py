from contextlib import contextmanager
from dataclasses import dataclass, field
import sys
import argparse
import threading
import traceback

from concurrent.futures import ThreadPoolExecutor

from time import sleep, time
from typing import List, Optional
from urllib.parse import urlparse
from uuid import uuid4
from pathlib import Path

import requests
import httpx

from scipy.stats import poisson
from numpy.random import RandomState
from numpy import int32

from runner import workers

@dataclass
class Result:
    uid: str
    start_t: float
    end_t: float

class QueueRunner:
    runner_id: str
    experiment_id: str
    url: str
    hostname: str
    port: int
    reqs_limit: int
    timeout: Optional[float]

    daemons: List[threading.Thread]
    executor: ThreadPoolExecutor

    def __init__(self, args: argparse.Namespace):
        self.runner_id = args.runner_id or str(uuid4())
        self.experiment_id = args.experiment_id or str(uuid4())
        self.gzip = args.gzip

        self._state = RandomState(args.seed + hash(self.runner_id) % 10000 + 128)

        self.url = args.url
        if not self.url.startswith('http'):
            self.url = f'http://{self.url}'

        self.body_type = args.body_type
        self.bodies = args.bodies
        self._init_bodies()

        self.method = args.method
        self.hostname = args.hn or urlparse(self.url).hostname
        self.port = urlparse(self.url).port

        if args.t is not None and args.n == -1:
            self.reqs_limit = sys.maxsize
        else:
            self.reqs_limit = args.n

        self.timeout = args.t

        self.daemons = [threading.Thread(target=workers.timeout_worker, args=(self.timeout,), daemon=True)]

    
    def _init_bodies(self):
        bodies: List[str] = self.bodies

        nbodies = []

        try:
            if bodies is None or len(bodies) == 0:
                return

            cwd = Path.cwd()

            for body in bodies:
                if body.startswith('@'):
                    with open(cwd / body[1:], 'rb') as f:
                        nbodies.append(f.read())
                else:
                    nbodies.append(body)
        finally:
            self.bodies = nbodies


    def next_body(self):
        if len(self.bodies) == 0:
            return None

        return self.bodies[self._state.choice(len(self.bodies))]


    def run(self, *args, **kwargs):
        workers.setup(
            url=self.url,
            method=self.method,
            hostname=self.hostname,
            port=self.port,
            body_type=self.body_type,
            reqs_limit=self.reqs_limit
        )

        workers.reset()

        self.executor = ThreadPoolExecutor(*args, **kwargs)

        for t in self.daemons:
            t.start()

class ConcurrentQueueRunner(QueueRunner):
    '''
    Naive implementation where there are T threads making requests constantly.
    '''

    delay: float
    concurrency: int

    def __init__(self, args: argparse.Namespace):
        super().__init__(args)

        self.delay = args.d
        self.concurrency = args.c

    def next_delay(self):
        return self.delay

    @contextmanager
    def run(self):
        super().run(max_workers=self.concurrency)

        worker_options = {
            'type': 'concurrent',
            'concurrency': self.concurrency,
            'gzip': self.gzip
        }

        with requests.Session() as session:
            try:
                try:
                    args = []
                    args.append([self.experiment_id] * self.concurrency)
                    args.append([worker_options] * self.concurrency)
                    args.append([self.next_delay] * self.concurrency)
                    args.append([self.next_body] * self.concurrency)
                    args.append([session] * self.concurrency)

                    total_start_t = time()
                    list(self.executor.map(workers.request_worker, *args))
                    total_end_t = time()
                except KeyboardInterrupt:
                    total_end_t = time()
                    print('')
                    workers._stop_ev.set()

                # print('')
                # print(f'Finished {len(results)} requests')
            finally:
                in_flight_reqs = workers._reqs_current - len(workers.results)

                print('')
                print('Concurrent workers queue runner:')
                print(f' + delay: {self.delay}')
                print(f' + concurrency: {self.concurrency}')
                print(f' * in-flight request(s): {in_flight_reqs}')
                print(f' * executor queue: {self.executor._work_queue.qsize()}')
                print(f' * executor threads: {len(self.executor._threads)}')

                if in_flight_reqs:
                    print('')
                    print('Waiting for requests to finish...')

                self.executor.shutdown(wait=False, cancel_futures=True)
                yield Result(self.experiment_id, total_start_t, total_end_t)
        


class PoissonQueueRunner(QueueRunner):
    qtype = 'default'

    def __init__(self, args: argparse.Namespace):
        super().__init__(args)

        self._state = RandomState(args.seed + hash(self.runner_id) % 10000 + 121)
        self._max_workers = args.max_concurrency if args.max_concurrency != -1 else 2**20
        self.max_workers = self._max_workers

        self.mean_req_time = int(1e6 / args.max_throughput)
        self.max_throughput = args.max_throughput
        self.max_concurrency = args.max_concurrency

        self.daemons.append(threading.Thread(target= self.slow_start, args=[], daemon=True))

    def next_delay(self):
        return poisson.rvs(self.mean_req_time, random_state=self._state) / 1e6

    def slow_start(self):
        self._max_workers = 1

        while not workers._stop_ev.is_set():
            sleep(1)

            if self._max_workers >= self.max_workers:
                break

            if not workers._ready_ev.is_set():
                workers._ready_ev.wait()

            self._max_workers += 1

    @contextmanager
    def run(self):
        super().run(max_workers=self.max_workers)

        worker_options = {
            'id': self.runner_id,
            'type': f'poisson_{self.qtype}',
            'throughput': self.max_throughput,
            'concurrency': self.max_concurrency,
            'gzip': self.gzip,
            'state': None
        }

        with requests.Session() as session:
            try:
                try:
                    total_start_t = time()
                    for _ in range(self.reqs_limit):
                        in_flight_reqs = workers._reqs_current - len(workers.results)

                        worker_options['state'] = {
                            'in_flight_requests': in_flight_reqs,
                            'mean_request_time': self.mean_req_time,
                            'workqueue_threads': self.executor._work_queue.qsize(),
                            'total_threads': len(self.executor._threads),
                            'idle_threads': self.executor._idle_semaphore._value,
                        }

                        # wait for a free thread before going
                        if in_flight_reqs >= self._max_workers:
                            workers._ready_ev.clear()
                            workers._ready_ev.wait()

                        if workers._stop_ev.wait(self.next_delay()):
                            break
                        
                        self.executor.submit(workers.request_worker, self.experiment_id, dict(worker_options), None, self.next_body, session)

                    workers._stop_ev.wait()
                    total_end_t = time()
                except KeyboardInterrupt:
                    total_end_t = time()
                    print('')
                    workers._stop_ev.set()

                # print('')
                # print(f'Finished {len(results)} requests')
                    
            finally:
                in_flight_reqs = workers._reqs_current - len(workers.results)

                print('')
                print('Poisson queue runner:')
                print(f' + type: {self.qtype}')
                print(f' + max throughput: {self.max_throughput}')
                print(f' + max concurrency: {self.max_concurrency}')
                print(f' * in-flight request(s): {in_flight_reqs}')
                print(f' * executor queue: {self.executor._work_queue.qsize()}')
                print(f' * executor threads: {len(self.executor._threads)}')

                if in_flight_reqs:
                    print('')
                    print('Waiting for requests to finish...')

                self.executor.shutdown(wait=False, cancel_futures=True)
                yield Result(self.experiment_id, total_start_t, total_end_t)



class SustainedPoissonQueueRunner(PoissonQueueRunner):
    qtype = 'sustained'

    def __init__(self, args: argparse.Namespace):
        super().__init__(args)

        self.daemons.append(threading.Thread(target= self.sustain_throughput, args=[], daemon=True))

    def run(self):
        self.max_workers = 128

        return super().run()

    def sustain_throughput(self):
        while not workers._stop_ev.is_set():
            sleep(1)

            idle_threads = self.executor._idle_semaphore._value
            in_flight_req = workers._reqs_current - len(workers.results)
            workqueue_size = self.executor._work_queue.qsize()

            if self.max_concurrency > 0 and in_flight_req >= self.max_concurrency:
                self.mean_req_time *= 1.0204

            sleep(0.5)

            # print('debug sustain          ', idle_threads, in_flight_req)

            # decrease the mean request time by 2% if there are idle threads

            new_idle_threads = self.executor._idle_semaphore._value

            # print('debug sustain idle     ', idle_threads, new_idle_threads)

            if new_idle_threads >= idle_threads:
                self.mean_req_time *= 0.98

            # increase the mean request time by 2% if the in-flight requests are rising

            new_in_flight_req = workers._reqs_current - len(workers.results)

            # # print('debug sustain in-flight', in_flight_req, new_in_flight_req)
            # # print('debug sustain workqueue', workqueue_size, new_workqueue_size)

            new_workqueue_size = self.executor._work_queue.qsize()

            if in_flight_req < new_in_flight_req or workqueue_size < new_workqueue_size:
                self.mean_req_time *= 1.0204
            
            # print('debug sustain mean', int(1e9 / self.mean_req_time) / 1e3)



class VariablePoissonQueueRunner(PoissonQueueRunner):
    qtype = 'variable'

    def __init__(self, args: argparse.Namespace):
        super().__init__(args)

        self.daemons.append(threading.Thread(target= self.vary_throughput, args=[], daemon=True))

    def vary_throughput(self):
        # TODO
        pass



class LinearIncreasePoissonQueueRunner(PoissonQueueRunner):
    qtype = 'linear-increase'

    def __init__(self, args: argparse.Namespace):
        super().__init__(args)

        self.min_throughput = args.min_throughput
        self.mean_req_time = int(1e6 / self.min_throughput)

        self.t_beginning = time()
        self.t_start = self._parse_time(args.t_start)
        self.t_end = self._parse_time(args.t_end)

        self.daemons.append(threading.Thread(target= self.linear_increase, args=[], daemon=True))

    def _parse_time(self, t: str) -> float:
        if not t:
            return 0
        
        if self.timeout is not None:
            if t.endswith('%'):
                return float(t[:-1]) / 100.0 * self.timeout
            else:
                return float(t)
        else:
            if t.endswith('%'):
                return int(float(t[:-1]) / 100.0 * self.reqs_limit)
            else:
                return int(t)
        

    def linear_increase(self):
        while not workers._stop_ev.is_set():
            sleep(1)

            if self.timeout is not None:
                t_current = time() - self.t_beginning
            else:
                t_current = workers._reqs_current

            if t_current < self.t_start:
                pass
            elif t_current >= self.t_end:
                self.mean_req_time = int(1e6 / self.max_throughput)
                break
            else:
                # r_left = t_current - self.t_start
                # r_right = self.t_end - t_current
                # r_left = r_left / (r_left + r_right)
                # r_right = r_right / (r_left + r_right)

                # self.mean_req_time = int(1e6 / (self.min_throughput * r_right + self.max_throughput * r_left))

                t_diff = t_current - self.t_start
                t_total = self.t_end - self.t_start

                th_diff = self.max_throughput - self.min_throughput

                self.mean_req_time = int(1e6 / (self.min_throughput + th_diff * t_diff / t_total))

