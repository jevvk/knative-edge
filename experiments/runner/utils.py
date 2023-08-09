from numbers import Number
from typing import List

import math
import argparse

def percentile(N, percent, key=lambda x:x):
    if not N:
        return None
    k = (len(N)-1) * percent
    f = math.floor(k)
    c = math.ceil(k)
    if f == c:
        return key(N[int(k)])
    d0 = key(N[int(f)]) * (c-k)
    d1 = key(N[int(c)]) * (k-f)
    return d0+d1

class percentile:
    def __init__(self, data: list, key=None):
        self.data = sorted(data, key=key)
    
    def get(self, percent: int, key=lambda x:x):
        if not self.data:
            return None
        k = (len(self.data)-1) * percent
        f = math.floor(k)
        c = math.ceil(k)
        if f == c:
            return key(self.data[int(k)])
        d0 = key(self.data[int(f)]) * (c-k)
        d1 = key(self.data[int(c)]) * (k-f)
        return d0+d1
  
def graph(args: argparse.Namespace, msg: str, T: List[Number]):
    buckets = [0] * min(args.graph_width, len(T))
    left = len(T)
    offset = 0
    height = args.graph_height

    for b in range(len(buckets)):
        nrun = (left - 1) // (len(buckets) - b) + 1
        buckets[b] = sum(T[offset:offset+nrun]) / nrun
        offset += nrun
        left -= nrun

    bmax = max(buckets)
    for b in range(len(buckets)):
        buckets[b] = int(buckets[b] * height / bmax)

    print(msg)
    print('')

    print(f'time ({bmax*1000:.2f} ms)')
    for i in range(height, 0, -1):
        print(' |', end='')
        for j in range(len(buckets)):
            if buckets[j] >= i:
                print('.', end='')
            else:
                print(' ', end='')
        print('')
    print(' +' + '-' * len(buckets) + f' request # ({len(T)})')