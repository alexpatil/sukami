# Engineering & Code Quality Standards

These rules define the design principles, structural rules, and coding standards for all code in this repository.

## 1. Core Principles
* **KISS (Keep It Simple, Stupid)**: Favor simple, clear, readable solutions over clever abstractions.
* **YAGNI (You Aren't Gonna Need It)**: Implement only what is required right now. Do not build speculative features.
* **DRY (Don't Repeat Yourself)**: Extract common logic into reusable functions or packages without introducing premature abstractions.
* **SOLID Principles**:
  * **Single Responsibility**: Each file and package must have one clear purpose.
  * **Open/Closed**: Design modules to be extensible via interfaces.
  * **Liskov Substitution**: Types implementing interfaces must fulfill the full contract.
  * **Interface Segregation**: Keep interfaces small, focused, and client-driven.
  * **Dependency Inversion**: High-level modules should depend on abstractions (interfaces), not concrete types.

## 2. File & Size Constraints
* **Strict Line Limit**: No source file should exceed **250 lines of code**. If a file approaches 250 lines, split it into cohesive, single-responsibility files.
* **Function Length**: Functions should ideally be under **50 lines** and do one thing well.

## 3. Go Idioms & Best Practices
* Use Go standard library wherever possible (`net/http`, `log/slog`, `sync`, `time`, `context`).
* Structured logging with `log/slog` for zero-allocation and structured output.
* Explicit error handling: Never ignore errors. Return wrapped context where helpful.
* Graceful concurrency with channels, `sync.Mutex`/`RWMutex`, and `context.Context` propagation.
* Zero external bloat: Keep dependencies minimal, secure, and fast.

## 4. Web & UI Standards
* Premium dark-mode aesthetics with CSS variables and glassmorphism.
* Responsive layouts, fluid micro-interactions, and instant feedback.

## 5. Mandatory Type-Checking & Pre-Build Analysis
* **Static Analysis**: Always execute `go vet ./...` to detect type discrepancies, invalid format specifiers, unreachable code, and suspicious constructs.
* **Pre-Build Verification**: Validate that all types conform to interface signatures (`proxy.Cache`, `proxy.RequestTracker`) before triggering any build.
* **Zero Warnings Policy**: Unused imports, unused variables, and type conversion errors must be resolved immediately without exceptions.
