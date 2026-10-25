# Contributing to Sukami

Thank you for your interest in contributing to **Sukami**! We welcome contributions from developers worldwide to help make LLM token optimization faster, leaner, and more accessible.

To keep the project maintainable, blazingly fast, and high quality, please review the following guidelines before submitting a pull request.

---

## 🧭 Core Principles

1. **Zero Bloat & Stdlib First**: Avoid heavy external frameworks or unnecessary third-party dependencies. If it can be done cleanly with standard Go libraries, prefer stdlib.
2. **KISS & YAGNI**: Keep designs minimal, straightforward, and performant. Avoid premature over-engineering.
3. **Strict Modularity (Max 250 Lines/File)**: Every source file must remain under **250 lines of code**. If a file grows beyond 250 lines, split it into cohesive, single-responsibility files.
4. **Mandatory Verification**: Every pull request must pass `go vet ./...` and `go test ./...` with zero errors or warnings.

---

## 🛠️ Getting Started

### Prerequisites
- **Go 1.22+** installed on your system.
- Git.

### Local Setup
1. **Fork** the repository on GitHub: [https://github.com/alexpatil/sukami](https://github.com/alexpatil/sukami)
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/sukami.git
   cd sukami
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/alexpatil/sukami.git
   ```
4. **Run the local gateway & embedded dashboard**:
   ```bash
   go run ./cmd/sukami
   ```
   Open `http://localhost:8080` in your browser to verify the dashboard and proxy.

---

## 🌿 Branching & Development Workflow

To ensure stability, direct pushes to `main` are restricted. All contributions follow the standard GitHub Flow:

1. **Sync with latest upstream**:
   ```bash
   git checkout main
   git pull upstream main
   ```
2. **Create a dedicated topic branch**:
   Use descriptive branch names with appropriate prefixes:
   - `feat/feature-name` for new capabilities
   - `fix/bug-description` for bug fixes
   - `perf/optimization-area` for performance improvements
   - `docs/topic-name` for documentation updates
   ```bash
   git checkout -b feat/add-anthropic-streaming
   ```
3. **Make your changes** following the coding standards below.
4. **Run pre-commit checks locally**:
   ```bash
   # Run static analysis
   go vet ./...

   # Run tests with race detection
   go test -v -race ./pkg/...

   # Verify compilation
   go build -v -o bin/sukami ./cmd/sukami
   ```

---

## 📝 Commit Message Convention

We follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

- `feat:` A new feature or capability (e.g., `feat(proxy): add support for custom provider headers`)
- `fix:` A bug fix (e.g., `fix(cache): prevent entity collision for similar technical terms`)
- `perf:` Performance improvements (e.g., `perf(optimizer): optimize BPE token counting loop`)
- `docs:` Documentation only changes (e.g., `docs(readme): add docker deployment instructions`)
- `test:` Adding missing tests or correcting existing tests (e.g., `test(auth): add rate limiter edge case tests`)
- `refactor:` A code change that neither fixes a bug nor adds a feature

---

## 🚀 Submitting a Pull Request (PR)

1. **Push your branch to your fork**:
   ```bash
   git push origin feat/your-feature-name
   ```
2. Open a Pull Request on GitHub against the `main` branch.
3. Fill out the Pull Request template completely.
4. Ensure all automated CI checks pass (`Go Backend Test & Vet`).
5. A maintainer will review your code. Be open to feedback and discussion!

---

## 📋 Code Standards Checklist

Before submitting your PR, confirm:
- [ ] No file exceeds **250 lines** of code.
- [ ] `go vet ./...` reports zero issues.
- [ ] `go test -v -race ./pkg/...` passes.
- [ ] Code is formatted with `go fmt ./...`.
- [ ] No secrets or personal API keys are committed.
- [ ] Unit tests are included for new features or bug fixes.

---

## 🤝 Community

We are committed to providing a friendly, safe, and welcoming environment for all. Please read and adhere to our [Code of Conduct](CODE_OF_CONDUCT.md).
