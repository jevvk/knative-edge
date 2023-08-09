from dataclasses import dataclass
import json
from random import random
from typing import List

from numbers import Number
import sys
from typing import List
import requests
import argparse
import threading
from urllib.parse import urlparse
from time import time, sleep
from concurrent.futures import ThreadPoolExecutor
import traceback
import elasticsearch
import uuid
from datetime import datetime
import pytz

from runner import workers, elastic
from runner.utils import percentile, graph
from runner.queue import Result


class Reporter:
    def __init__(self, args: argparse.Namespace):
        self.args = args

    def print(self, res: Result):
        # print(f'Finished {len(results)} requests')
        print('')

        if workers.last_response is None:
            print('No requests were made.')
            
            return

        total_content_length = 0

        failed_reqs = 0
        non2xx_req = 0

        reqs_t = []
        t_reqs_t = 0
        names = {}

        for result in workers.results:
            t = result.end_t - result.start_t

            if result.resp_status_code == -1:
                failed_reqs += 1
            if result.resp_status_code >= 300:
                non2xx_req += 1

            if result.resp_status_code != -1:
                reqs_t.append(t)
                t_reqs_t += t
                total_content_length += result.resp_content_length
            
            names[result.resp_server_name] = names.get(result.resp_server_name, 0) + 1

        reqs_t = sorted(reqs_t)
        names = sorted([ (v, k) for k, v in names.items() ], reverse=True)

        if len(names) > 5:
            names = ', '.join(map(lambda x: f'{x[1]} ({x[0]})', names[:5])) + f', +{len(names)-5} other'
        else:
            names = ', '.join(map(lambda x: f'{x[1]} ({x[0]})', names))

                #     last_err = req
                #     # stop_ev
        avg_t = t_reqs_t / len(workers.results)
        avg_t_total = (res.end_t-res.start_t) / len(workers.results)
        req_ps = len(workers.results) / (res.end_t-res.start_t)

        # print('')
        # print(last_response.headers)
        # print(f'Server software: {workers.last_response.headers.get("server", "unknown")}')
        print(f'Server hostname: {workers.hostname}')
        print(f'Server port:     {workers.port}')
        print(f'Server name(s):  {names}')

        print('')
        print(f'Concurrency level:   {self.args.c}')
        print(f'Time taken:          {res.end_t-res.start_t:.2f} seconds')
        print(f'Completed requests:  {len(workers.results)}')
        print(f'Failed requests:     {failed_reqs}')
        print(f'Non-2xx responses:   {non2xx_req}')
        print(f'Total transferred:   {total_content_length}')
        print(f'Requests per second: {req_ps:.2f} [#/sec] (mean)')
        print(f'Time per request:    {avg_t*1000:.2f} [ms] (mean)')
        print(f'Time per request:    {avg_t_total*1000:.2f} [ms] (mean, across all concurrent requests)')
        print(f'Transfer rate:       {total_content_length/(1000*(res.end_t-res.start_t)):.2f} [Kbytes/sec] received')

        perc = percentile(reqs_t)

        print('')
        print('Percentage of the requests served within a certain time (ms)')
        print(f' 50%  {int(1000*perc.get(0.5))}')
        print(f' 66%  {int(1000*perc.get(2.0/3))}')
        print(f' 75%  {int(1000*perc.get(0.75))}')
        print(f' 80%  {int(1000*perc.get(0.8))}')
        print(f' 85%  {int(1000*perc.get(0.85))}')
        print(f' 90%  {int(1000*perc.get(0.9))}')
        print(f' 95%  {int(1000*perc.get(0.95))}')
        print(f'100%  {int(1000*perc.get(1))}')

        if self.args.display_graphs:
            print('')
            graph(self.args, 'Response time graph:', list(map(lambda x: x[1]-x[0], workers.results)))

            print('')
            graph(self.args, 'Response time graph (ordered):', reqs_t)

        if self.args.out_resp:
            if workers.last_response is not None:
                print('')
                print('Last response:')
                print('response =', workers.last_response)
                print('elapsed =', workers.last_response.elapsed)
                print('headers =', json.dumps(dict(workers.last_response.headers), indent=2))

                text = workers.last_response.text
                if len(text) > 62:
                    text = text[:60] + '...'
                print('text =', text)

            if workers.last_err is not None:
                print('')
                print('Last non-2xx response:')
                print(workers.last_err)
                print(workers.last_err.headers)
                print(workers.last_err.text)

        if workers.last_exc is not None:
            print('Last worker exception:')
            traceback.print_exception(*workers.last_exc)

    def push(self, res: Result):
        print('')
        print(f'Flushing to elastic... [{res.uid}]')

        elastic.flush()

        print('Done.')
