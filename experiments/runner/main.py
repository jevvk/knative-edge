from runner.report import Reporter
from runner.queue import QueueRunner, ConcurrentQueueRunner, PoissonQueueRunner, SustainedPoissonQueueRunner, VariablePoissonQueueRunner, LinearIncreasePoissonQueueRunner

def main(args):
    if args.with_workers:
        runner: QueueRunner = ConcurrentQueueRunner(args)
    elif args.with_poisson:
        runner: QueueRunner = PoissonQueueRunner(args)
    elif args.with_poisson_sustained:
        runner: QueueRunner = SustainedPoissonQueueRunner(args)
    elif args.with_poisson_variable:
        runner: QueueRunner = VariablePoissonQueueRunner(args)
    elif args.with_poisson_linear_increase:
        runner: QueueRunner = LinearIncreasePoissonQueueRunner(args)
    else:
        print('No valid runnner found.')
        exit(1)

    reporter = Reporter(args)

    print(f'Benchmarking {runner.hostname}...')

    with runner.run() as result:
        reporter.print(result)
        reporter.push(result)

