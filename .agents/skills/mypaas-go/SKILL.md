---
name: mypaas-go
description: "Use when writing or refactoring Go code in the MyPaas backend to ensure professional, idiomatic, and consistent style."
---

# MyPaas Go Professional Standard

This skill defines the professional standards and architecture rules for all Go code written in the MyPaas backend. ALWAYS follow these guidelines when creating new features, refactoring, or auditing the codebase.

## 1. Architecture & Project Structure
- **Package-by-Feature:** Group code by domain feature (e.g., `deployment`, `container`, `project`), NOT by architectural layer (avoid `controllers`, `services`, `repositories` folders).
- **Encapsulation:** Place domain packages inside `internal/` so they cannot be imported by external modules.
- **Dependency Injection:** No global variables or `init()` functions for state. Use struct constructors (`NewService`, `NewHandler`) and pass dependencies (like database connections or other services) explicitly.

## 2. Naming Conventions
- **Packages:** Short, lowercase, single-word names (`deployment`, `auth`). Avoid generic names like `common`, `util`, `helpers` unless strictly necessary.
- **Interfaces:** Single-method interfaces should use the `-er` suffix (e.g., `Reader`, `Deployer`). Multi-method interfaces should have descriptive names.
- **Structs/Functions:** `PascalCase` for exported, `camelCase` for unexported. Keep acronyms in the same case (e.g., `APIURL`, not `ApiUrl`).
- **Errors:** Sentinel errors must start with `Err` (e.g., `ErrNotFound`).

## 3. Error Handling
- **Domain Errors:** Define all domain-specific sentinel errors in `internal/errs/errs.go`.
- **Wrapping:** ALWAYS wrap errors with context using `fmt.Errorf("do something: %w", err)` when passing them up the stack.
- **HTTP Translation:** Do not write HTTP response logic inside domain services. Services return `error`. The HTTP Handler uses `httpx.DomainError(w, err)` to translate domain errors into standardized JSON HTTP responses.
- **No Panics:** Never use `panic()` in business logic. Only use panic for unrecoverable startup configuration errors.

## 4. Context & Concurrency
- **Context is King:** Every function that performs I/O (Database, Docker CLI, HTTP calls) MUST take `ctx context.Context` as its first parameter and respect context cancellation.
- **Goroutines:** When spawning a goroutine, always consider how and when it will exit to avoid goroutine leaks. Avoid launching unmanaged "fire and forget" goroutines without a WaitGroup or context propagation unless it's a dedicated background worker.

## 5. Clean Code Practices
- **Small Functions:** Keep functions under 40-50 lines. If a function grows larger, split the logic into helper methods.
- **Parameter Lists:** Keep parameters to a maximum of 4 or 5. If more are needed, introduce an `Options` struct.
- **Defer:** Always use `defer` to close resources (files, bodies, connections) immediately after successfully opening them.
- **Type Safety:** Avoid `interface{}` or `any` unless absolutely required (like generic JSON decoding). Use Go Generics (`[T any]`) instead.

## 6. Database & SQL
- **SQLC Only:** Do not use GORM, sqlx, or raw `database/sql` strings in domain logic. All database access must go through `sqlc`-generated code in `internal/db`.
- **Query Organization:** Write raw SQL in `backend/query/` and run `make sqlc` to generate the type-safe wrappers.

## 7. Logging
- **Structured Logs:** Use the standard library `log/slog`. Do not use `logrus` or `zap`.
- **Traceability:** Include identifiers in logs (e.g., `projectId`, `userId`) but NEVER log sensitive information (tokens, passwords, environment variable values).
