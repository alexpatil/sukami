# Project Instructions: Sukami AI Token Proxy

## Core Directives
1. **KISS & YAGNI**: Keep the architecture straightforward and minimal. Avoid unnecessary complexity.
2. **DRY**: Reuse common utilities (token estimation, JSON response helpers, time formatters).
3. **SOLID**:
   - High-level handlers rely on interfaces (`Cache`, `Tracker`, `Optimizer`).
   - Single responsibility per file.
4. **File Length Limit**: **Strict maximum 250 lines per file**. Always split files if they grow beyond 250 lines.
5. **Language Standards**: Idiomatic Go 1.27+, stdlib first, zero unnecessary dependencies.
6. **Mandatory Type-Checking & Pre-Build Verification**:
   - Always run static type checking and vet analysis (`go vet ./...`) before finishing any modification.
   - Catch and fix any type errors, unused imports, or signature mismatches immediately before building or deploying.
