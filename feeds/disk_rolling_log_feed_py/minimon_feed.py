import os
import time
import sys

class MiniMonFeed:
    def __init__(self, output_dir):
        self.output_dir = output_dir
        self.pid = os.getpid()
        self.start_log_path = os.path.join(output_dir, "start.log")
        self.run_log_path = os.path.join(output_dir, f"{int(time.time())}_{self.pid}_run.log")
        self.stop_log_path = os.path.join(output_dir, "stop.log")

    def _write_log(self, path, message):
        with open(path, 'a', buffering=1) as f:
            print(f"{time.strftime('%Y-%m-%d %H:%M:%S')}: {message}\n")
            f.write(f"{time.strftime('%Y-%m-%d %H:%M:%S')}: {message}\n")

    def start_log(self):
        self._write_log(self.start_log_path, f"{self.pid}: Program started")

    def run_log(self):
        try:
            self._write_log(self.run_log_path, f"Start logging: {self.pid}")
            i = 0
            while True:
                self._write_log(self.run_log_path, f"INFO: Running log message {i + 1}")
                time.sleep(10)
                i += 1
        finally:
            self._write_log(self.run_log_path, f"End logging: {self.pid}")

    def stop_log(self):
        self._write_log(self.stop_log_path, f"End logging {self.pid}: Program stopped")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python minimon_feed.py <output_dir>")
        sys.exit(1)

    output_dir = sys.argv[1]
    feed = MiniMonFeed(output_dir)

    try:
        feed.start_log()
        feed.run_log()
    finally:
        feed.stop_log()
