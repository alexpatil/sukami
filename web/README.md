# 🌐 Sukami Embedded Web Dashboard

This directory contains the self-contained developer status dashboard for the Sukami AI token optimization gateway.

## Overview
- **Zero Build Step**: Built with pure HTML5, CSS3, and Vanilla JavaScript. No Node.js, npm, webpack, or external bundlers required.
- **Embedded Static Asset**: Directly embedded into the Go binary at build time via Go standard library's `embed.FS` (`//go:embed web/*`).
- **Real-Time Telemetry**: Visualizes total requests, semantic cache hits, prompt token savings, and estimated USD cost reductions.
- **Interactive Sandbox**: Test transparent prompt pruning, live model routing, and token analytics directly from the browser at `http://localhost:8080`.
