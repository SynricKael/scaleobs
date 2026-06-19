# Changelog

All notable changes to ScaleObs are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

---

## [v0.2.0] - 2026-06-19

### Fixed
- Tauri desktop client no longer shows blank screen: WebView now loads `localhost:8080` (gateway-served dashboard) instead of Vite dev server
- Tauri client automatically spawns the gateway process on startup — no more manual `gateway-release.exe` launch required
- TypeScript build errors in `ServerSettingsDialog.vue` (tab key type mismatch, missing `agent.type` field)
- Gateway binary path resolution in Tauri Rust backend (was looking at project root, now points to `gateway/gateway-release.exe`)
- Headscale sync now correctly fills `name` and `host` fields for agent-connected servers

### Changed
- Binary renamed from `sop-client.exe` to `scaleobs.exe`
- Window title updated to "ScaleObs"
- Cargo package name and tauri `productName` updated to `ScaleObs`
- Bundle installer filenames updated (MSI/NSIS)

---

## [v0.1.0] - 2026-06-19

### Added
- Initial public release
- Go-based Gateway server with REST API + WebSocket agent transport
- Lightweight system agent (Linux/macOS/Windows) collecting CPU, memory, disk, network, Docker metrics
- Headscale auto-discovery: pull server nodes from one or more Headscale networks
- Remote Docker monitoring via TCP API
- Coding agent auto-detection (opencode, codex, claude code) on monitored hosts
- AI Agent Server panel with manual add support (Basic Auth)
- Tauri 2 + Vue 3 desktop dashboard with server cards, gauge meters, Docker inline controls
- Server settings dialog (SSH, group assignment, AI Agent link)
- Per-server group filtering and YAML config management
- Agent binary distribution (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- Day/night theme toggle with localStorage persistence
- Bilingual README (EN/ZH)
