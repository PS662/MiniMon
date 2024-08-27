
# MiniMon

MiniMon is a lightweight log monitoring tool that tracks changes in specified sources (directories, files, or processes) and provides summary notifications within a configurable interval.


### Key Components

- **`config.json`**: Configuration for monitoring sources, log directory, and notification intervals.
- **`feeds/rolling_log/minimon_feed.py`**: Example Python script that generates simulated logs.
- **`minimon.go`**: Main Go program for monitoring sources.
- **`go.mod`** and **`go.sum`**: Go module files for dependency management.

### Usage

1. **Run the Python Log Generator**:
    ```bash
    cd feeds/rolling_log/
    python3 minimon_feed.py <out_log_dir>
    ```

2. **Run the Go Monitoring Program**:
    ```bash
    cd MiniMon
    go run minimon.go
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
    ./install.sh <user> <group>
    ```

Logs can be viewed with:

```bash
journalctl -u minimon.service
```