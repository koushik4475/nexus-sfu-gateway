# Nexus SFU Gateway | Go WebRTC Selective Forwarding Unit for Real-Time Video

Nexus SFU Gateway is a Go-based WebRTC media server project built around a Selective Forwarding Unit (SFU) architecture. It provides a starting point for exploring real-time audio/video routing, peer connections, and media gateway services using the Pion WebRTC ecosystem. The README covers installation, health checks, repository structure, and automated test commands.


<p align="center">
  <img src="https://capsule-render.vercel.app/api?type=waving&color=gradient&customColorList=2,14,26&height=220&section=header&text=NEXUS%20SFU%20GATEWAY&fontSize=42&fontColor=fff&animation=twinkling&desc=Zero-Latency%20Audio-Visual%20Streaming%20Gateway%20with%20WebRTC%20SFU%20in%20Go&descSize=16&descAlignY=68" width="100%" alt="Nexus SFU Gateway Header" />
</p>

<p align="center">
  <a href="https://github.com/ER-brocode/nexus-sfu-gateway/actions"><img src="https://img.shields.io/github/actions/workflow/status/ER-brocode/nexus-sfu-gateway/ci.yml?branch=main&style=for-the-badge&logo=github-actions&logoColor=white&label=CI%20Pipeline" alt="CI Status" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" /></a>
  <a href="https://webrtc.org"><img src="https://img.shields.io/badge/WebRTC-Pion%20v3-323330?style=for-the-badge&logo=webrtc&logoColor=white" alt="Pion WebRTC" /></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge" alt="License" /></a>
</p>

---

## 🌟 Overview

**Nexus SFU Gateway** is an ultra-low-latency Selective Forwarding Unit (SFU) audio-visual streaming gateway written in pure Go. Engineered for distributed media applications, high-concurrency video conferencing, virtual events, and interactive robotics, Nexus routes RTP/RTCP streams directly between participants without transcoding overhead.

```
       +-----------------------+
       |   Publisher (Alice)   |
       +-----------+-----------+
                   | Ingress RTP/RTCP (Opus, VP8, H.264)
                   v
       +-----------------------------------------------------------+
       |                   NEXUS SFU GATEWAY                       |
       |  +---------------------+        +----------------------+  |
       |  | Adaptive Jitter Buf | -----> | Fan-Out Forwarder    |  |
       |  +---------------------+        +-----------+----------+  |
       |             |                               |             |
       |             v                               v             |
       |  +---------------------+        +----------------------+  |
       |  |  GCC Congestion BWE |        | NACK / PLI Rewriter  |  |
       |  +---------------------+        +----------------------+  |
       +---------------------------------------------+-------------+
                         |                           |
         Downlink 1      |                           |  Downlink 2
                         v                           v
             +-----------------------+   +-----------------------+
             |   Subscriber (Bob)    |   |   Subscriber (Carol)  |
             +-----------------------+   +-----------------------+
```

---

## ✨ Architectural Features

- **Selective Forwarding Unit Core**: Zero-copy packet fan-out engine with non-blocking egress ring buffers.
- **Adaptive Jitter Buffer**: Automatically buffers and re-sequences out-of-order packets while mitigating playout delay.
- **Real-Time RTCP Feedback Controller**:
  - **NACK Generator**: Identifies missing sequence numbers and generates RFC 4585 loss retransmissions.
  - **PLI / FIR Throttling**: Coordinates keyframe refresh across subscriber track changes without publisher flood.
- **Google Congestion Control (GCC) AIMD**: Dynamically adjusts bandwidth allocation and downscale thresholds according to RTCP receiver reports and REMB feedback.
- **RTP Payload Depacketizer**: Real-time packet parsing and keyframe boundary detection for **VP8**, **Opus**, and **H.264**.
- **Real-Time WebSocket Signaling Server**: Room management and session negotiation supporting concurrent rooms and auto-reconnection.

---

## 📦 Repository Structure

```
nexus-sfu-gateway/
├── .github/
│   └── workflows/
│       └── ci.yml               # Automated CI pipeline (lint, vet, race, test, build)
├── cmd/
│   └── server/
│       └── main.go              # Production gateway binary entrypoint
├── pkg/
│   ├── rtp/
│   │   ├── depacketizer.go      # VP8, Opus, and H.264 RTP depacketizer & keyframe inspector
│   │   ├── jitter_buffer.go     # Adaptive jitter buffer with sequence gap tracking
│   │   └── rtcp_handler.go      # PLI, FIR, NACK, and REMB RTCP control loops
│   ├── sfu/
│   │   ├── bwe.go               # AIMD bandwidth estimator and congestion state machine
│   │   ├── forwarder.go         # Multi-subscriber RTP stream fan-out router
│   │   └── router.go            # WebRTC PeerConnection session manager & ingress pump
│   └── signaling/
│       └── server.go            # Gorilla WebSocket signaling server & Room manager
├── tests/
│   ├── room_lifecycle_test.go   # Integration tests for signaling, room joining, leave & offer exchange
│   └── stream_forwarding_test.go# Unit & integration tests for packet routing, jitter buffer, and BWE
├── go.mod                       # Pion WebRTC v3 & Gorilla WebSocket dependencies
├── go.sum                       # Cryptographic checksums
└── README.md                    # Project documentation & benchmarks
```

---

## 🚀 Quick Start

### Prerequisites
- Go 1.22 or higher
- Git

### Installation & Run

```bash
# Clone the repository
git clone https://github.com/ER-brocode/nexus-sfu-gateway.git
cd nexus-sfu-gateway

# Download dependencies
go mod download

# Run the gateway server
go run cmd/server/main.go --port 8080
```

### Health Check

```bash
curl http://localhost:8080/healthz
# Response: {"status":"healthy","gateway":"nexus-sfu-gateway"}
```

---

## 🧪 Running Automated Tests

```bash
# Run unit & integration test suites with race detection
go test -v -race ./...
```

---

## 🛡️ License

Distributed under the MIT License. See `LICENSE` for more information.
