import subprocess
import os
import sys
import tempfile
from typing import IO, List
import signal
from uuid import uuid4

def print_pipe(pipe: IO):
    data: bytes = pipe.read()

    if data is None:
        return

    print(data.decode())

if __name__ == "__main__":
    processes: List[subprocess.Popen] = []
    n_processes = int(sys.argv[1])
    p_args = sys.argv[2:]

    print("Benchmarking... (press Ctrl+C to stop)")

    with tempfile.TemporaryDirectory() as td:
        eid = str(uuid4())

        for _ in range(n_processes):
            args = ["python3", "-m", "runner", "--experiment-id", eid] + p_args
            process = subprocess.Popen(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            processes.append(process)

        try:
            for process in processes:
                process.wait()
        except KeyboardInterrupt:
            print("Stopping processes...")

            # for process in processes:
            #     if process.poll() is not None:
            #         continue

            #     process.send_signal(signal.CTRL_C_EVENT)

        finally:
            for process in processes:
                if process.poll() is not None:
                    continue

                process.wait()

            print("Done.")
            print("")
        
            for p, process in enumerate(processes):
                print("")
                print("STDOUT PROCESS", p)
                print("----------------")
                print_pipe(process.stdout)
                print("")

                print("STDERR PROCESS", p)
                print("----------------")
                print_pipe(process.stderr)
                print("")



