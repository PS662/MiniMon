
# MiniMon

MiniMon is a lightweight log monitoring tool that tracks changes in specified sources (directories, files, or processes) and provides summary notifications within a configurable interval.

## Project Structure

```plaintext
MiniMon/
├── config.json
├── feeds/
│   └── disk_rolling_log_feed_py/
│       ├── minimon_feed.py
│       └── temp_test/
├── go.mod
├── go.sum
├── minimon.go
└── testdata/
```

### Key Components

- **`config.json`**: Configuration for monitoring sources, log directory, and notification intervals.
- **`feeds/disk_rolling_log_feed_py/minimon_feed.py`**: Python script that generates simulated logs.
- **`minimon.go`**: Main Go program for monitoring sources.
- **`go.mod`** and **`go.sum`**: Go module files for dependency management.

### Usage

1. **Run the Python Log Generator**:
    ```bash
    cd feeds/disk_rolling_log_feed_py/
    python3 minimon_feed.py temp_test/
    ```

2. **Run the Go Monitoring Program**:
    ```bash
    cd MiniMon
    go run minimon.go
    ```

Sample Go output if there are invalid sources:
```plaintext
Invalid source: mon_src4 (/path/to/directory2)
Invalid source: mon_src2 (1234)
Invalid source: mon_src3 (/path/to/file.log)
```

### Setup

- Initialize Go module:
    ```bash
    go mod init minimon
    ```
- Install dependencies:
    ```bash
    go get github.com/fsnotify/fsnotify
    go get github.com/gen2brain/beeep
    ```

MiniMon monitors valid sources and provides notifications summarizing changes within each interval.

### Running MiniMon at Startup as a Background Process

You can run `minimon.go` at startup by following these steps:

1. Run the `install.sh` script to set up the service:

    ```bash
    chmod +x install.sh
    sudo ./install.sh
    ```

2. The script will:
    - Build the `minimon` binary.
    - Create a systemd service to run `minimon` at startup.
    - Enable and start the service automatically.

3. Check the status of the service:

    ```bash
    sudo systemctl status minimon.service
    ```

4. To remove run:

    ```
    chmod +x install.sh
    sudo ./install.sh
    ```

Logs can be viewed with:

```bash
journalctl -u minimon.service
```