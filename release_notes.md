# Changelog

## v1.0.1

### New Features
* **Storage quota reporting:** Optional `--quota` flag (e.g. `2G`, `512M`) so the WebDAV share reports a fixed capacity to clients. Used space is the size of the data directory; available space is quota minus used. Clients that support RFC 4331 (e.g. macOS Finder) will show the correct size. 

  *Note: Windows Explorer does not use server-reported quota and will still show the local C: drive size.*
* **README:** Added project README with features overview, commands, server flags, quick start, and configuration.

### Changes
* **Server:** New `QuotaBytes` field and `--quota` CLI/config/env (`ATLAS_QUOTA`) support. When set, PROPFIND responses inject `quota-available-bytes` and `quota-used-bytes` based on the quota and actual dir usage instead of the host filesystem.
* **Quota middleware:** Uses `getDirUsedBytes()` for quota-based reporting when `QuotaBytes > 0`; otherwise keeps previous filesystem-based disk usage (Linux) or stub (other OS).