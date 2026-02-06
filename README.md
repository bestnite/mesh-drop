# Mesh Drop

English | [中文](./README.zh.md)

Simple, fast LAN file transfer tool, built with Wails and Vue.

## Features

- **File Transfer**: Support multi-file sending, easily share.
- **Folder Transfer**: Support sending entire folder structures.
- **Text Transfer**: Quickly sync text content between devices.
- **Encrypted Transmission**: Ensure data security during transmission.
- **Secure Identity**: Ed25519-based signature verification to prevent spoofing.

## Security Mechanisms

Mesh Drop uses a multi-layered security design to protect users from potential malicious attacks:

1.  **Identity**
    - Each device generates a unique pair of Ed25519 keys on first startup.
    - All presence broadcasts are signed with the private key.
    - The receiver verifies the signature with the public key to ensure the identity has not been tampered with.

2.  **Trust**
    - Uses TOFU (Trust On First Use) strategy.
    - Users can choose to "Trust" a Peer. Once trusted, that Peer's public key is pinned.
    - Subsequent packets from that Peer ID must be verified by the saved public key, otherwise they will be marked as **Mismatch**.
    - **Anti-spoofing**: If someone tries to spoof a trusted Peer ID, the UI will display a clear "Mismatch" security warning and prevent metadata from being overwritten.

3.  **Encryption**
    - File transfer service uses HTTPS protocol.
    - Automatically generates self-signed certificates for communication encryption to prevent eavesdropping.

## Screenshots

| ![Mesh Drop](./screenshot/1.png) | ![Mesh Drop](./screenshot/2.png) |
| -------------------------------- | -------------------------------- |

## Todo

- [x] Clipboard transfer
- [x] Folder transfer
- [x] Cancel transfer
- [x] Multi-file sending
- [x] Encrypted transmission
- [x] Settings page
- [x] Single instance mode
- [x] System notifications
- [x] Clear history
- [x] Auto accept
- [x] App icon
- [x] Trust Peer
- [x] Multi-language support
- [ ] System tray (minimize to tray) badges https://github.com/wailsapp/wails/issues/4494

## Tech Stack

This project is built using a modern tech stack:

- **Backend**: [Go](https://go.dev/) + [Wails v3](https://v3.wails.io/)
- **Frontend**: [Vue 3](https://vuejs.org/) + [TypeScript](https://www.typescriptlang.org/)
- **UI Framework**: [Vuetify](https://vuetifyjs.com/)

## Development

### Prerequisites

Before starting, ensure your development environment has the following tools installed:

1.  **Go** (version >= 1.25)
2.  **Node.js**
3.  **Wails CLI**
4.  **UPX**

### Install Dependencies

```bash
# Enter project directory
cd mesh-drop

# Install frontend dependencies (Wails usually handles this automatically, but manual installation ensures a clean environment)
cd frontend
npm install
cd ..
```

### Run Development Environment

```bash
wails3 dev
```
