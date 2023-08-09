import argparse

parser = argparse.ArgumentParser()
parser.add_argument('--experiment-id', help='Optional experiment ID', type=str, required=False, default=None)
parser.add_argument('--runner-id', help='Optional runner ID', type=str, required=False, default=None)
parser.add_argument('url', help='URL to be benchmarked', type=str)
group = parser.add_mutually_exclusive_group(required=True)
group.add_argument('-n', help='number of requests', type=int, default=-1)
group.add_argument('-t', help='time limit', type=float, default=None)
parser.add_argument('--method', dest='method', help='request method', default='GET', choices=['GET', 'POST'])
parser.add_argument('--body', dest='bodies', help='request body (if multiple bodies are set, each requests will select one of the bodies at random)', nargs='*')
parser.add_argument('--body-type', dest='body_type', help='request body type (e.g. text/plain, application/json)', required=False, default=None)
parser.add_argument('--host', dest='hn', help='custom hostname', type=str, default=None, required=False)
parser.add_argument('--gzip', dest='gzip', help='enable gzip', action='store_true', default=False)
parser.add_argument('--graph', dest='display_graphs', help='print request graphs', action='store_true', default=False)
parser.add_argument('--graph-width', dest='graph_width', help='graph width', type=int, default=120)
parser.add_argument('--graph-height', dest='graph_height', help='graph height', type=int, default=20)
parser.add_argument('--output-response', dest='out_resp', help='print end request body', action='store_true', default=False)

group = parser.add_mutually_exclusive_group(required=True)
group.add_argument('--with-workers', help='make requests using X workers', dest='with_workers', action='store_true', default=False)
group.add_argument('--with-poisson', help='make requests using poisson distribution', dest='with_poisson', action='store_true', default=False)
group.add_argument('--with-poisson-variable', help='make requests using poisson distribution', dest='with_poisson_variable', action='store_true', default=False)
group.add_argument('--with-poisson-sustained', help='make requests using poisson distribution', dest='with_poisson_sustained', action='store_true', default=False)
group.add_argument('--with-poisson-linear-increase', help='make requests using poisson distribution with linear increase', dest='with_poisson_linear_increase', action='store_true', default=False)

group = parser.add_argument_group('with workers')
group.add_argument('-c', help='concurrency', type=int, required=False, default=1)
group.add_argument('-d', '--delay', dest='d', help='delay between requests', type=float, default=None, required=False)


group = parser.add_argument_group('with poisson')
group.add_argument('--seed', dest='seed', help='RNG seed', type=int, required=False, default=42)
group.add_argument('--max-throughput', dest='max_throughput', help='max throughput (in requests/second)', type=float, required=False, default=1)
group.add_argument('--max-concurrency', dest='max_concurrency', help='max number of in-flight request at any time', type=int, required=False, default=-1)

group = parser.add_argument_group('with poisson (linear increase)')
group.add_argument('--min-throughput', dest='min_throughput', help='min throughput (in requests/second)', type=float, required=False, default=0)
group.add_argument('--t-start', dest='t_start', help='until t-start, requests are sent at min-throughput (relative or absolute time)', type=str, required=False, default='0%')
group.add_argument('--t-end', dest='t_end', help='after t-end, requests are sent at max-throughput (relative or absolute time)', type=str, required=False, default='100%')

parser.add_argument('--elastic-host', dest='es_host', help='url of elasticsearch', type=str, required=True)
parser.add_argument('--elastic-user', dest='es_user', help='elasticsearch user', type=str, required=True)
parser.add_argument('--elastic-password', dest='es_pass', help='elasticsearch user\'s password', type=str, required=True)

args = parser.parse_args()

from runner import elastic
elastic.start(args)

from runner.main import main
main(args)

elastic.stop()
