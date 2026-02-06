# Atlas Storage

A high-performance, headless WebDAV server written in Go. Easily share files over HTTP and mount them as network drives on Windows, macOS, and Linux.

## Features

- **WebDAV Compliance**: Fully compatible with standard WebDAV clients (Windows Explorer, Finder, etc.).
- **User Management**: Built-in authentication (Basic Auth) with simple CLI management.
- **Quota Support**: Define storage limits which are correctly reported to the client OS.
- **Single Binary**: Deploys as a static binary or Docker container.

## Commands

- `atlas server [flags]` - Starts the WebDAV service
- `atlas user add <name> <pass>` - Adds a new user
- `atlas user rm <name>` - Removes an existing user
- `atlas user ls` - Lists all registered users

## Server Configuration

- `--port`, `-p` (Env: `ATLAS_PORT`)  
  Server listening port. Default: `8080`.

- `--data-dir`, `-d` (Env: `ATLAS_DATA_DIR`)  
  Root directory for file storage. Default: `./data`.

- `--quota` (Env: `ATLAS_QUOTA`)  
  Max storage size (e.g., `5GB`, `500MB`). Default: none.

## Quick Start

1. **Start Atlas**:
   ```bash
   atlas server --port 8080 --data-dir ./my-files --quota 10GB
   ```

2. **Add User**:
   ```bash
   atlas user add admin secret123
   ```

3. **Connect**:
   - **Windows**: Map Network Drive -> `http://<ip>:8080`
   - **macOS**: Finder -> Connect to Server -> `http://<ip>:8080`
   - **Linux**: `mount -t davfs http://<ip>:8080 /mnt/dav`
