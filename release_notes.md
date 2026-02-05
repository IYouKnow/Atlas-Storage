# Changelog v1.0.0

This is the initial release of Atlas Storage.

## New Features
* **Core WebDAV Server:** Implemented standard WebDAV compliant server (compatible with Windows Explorer mounting).
* **CLI Management:** Added `atlas user` commands (`add`, `ls`, `rm`) to manage access control without editing files manually.
* **Authentication:** Built-in Basic Auth middleware backed by a persistent `users.json` database.
* **Configuration:** Added support for Environment Variables (`ATLAS_PORT`, `ATLAS_DATA_DIR`) for 12-factor app compliance.

## Infrastructure & Build
* **Multi-Arch Support:** Cross-compilation enabled for Linux AMD64 (Servers) and ARM64 (Raspberry Pi).
* **Docker:** Included `Dockerfile` and `docker-compose.yml` for containerized deployment.
* **CI/CD:** Automated GitHub Actions pipeline for building and publishing releases.

## Assets
* `atlas-linux-amd64`: For standard Linux servers/VPS.
* `atlas-linux-arm64`: For Raspberry Pi and ARM servers.