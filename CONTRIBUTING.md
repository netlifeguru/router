# Contributing to netlifeguru/router

Hello! 👋 Thanks for your interest in contributing to the `router` project. Below are a few guidelines to help you get started and ensure consistency across contributions.

---

## 🐛 Reporting Issues

If you've found a bug or have a suggestion, please use the [GitHub Issues](https://github.com/netlifeguru/router/issues) tab.

When reporting an issue, please include:

- A clear description of the problem.
- Steps to reproduce the issue.
- What you expected to happen vs. what actually happened.
- Your environment (OS, Go version, etc.).

---

## 📦 Pull Requests

> **Note:** Please do **not** create pull requests directly against the `main` branch.  
> Always create a feature or fix branch first, such as `fix/route-matcher` or `feat/custom-routing`.

Pull requests should:

- Include tests for new functionality or bug fixes where applicable.
- Reference the related issue, if applicable.
- Be clearly named and scoped.
- Keep changes focused and avoid mixing unrelated updates.

---

## ✏️ Commit Messages

We follow a conventional commit format to keep the history clean and structured.  
Please prefix your commit messages with one of the following types:

- `feat:` – for new features
- `fix:` – for bug fixes
- `docs:` – for documentation changes, such as README or comments
- `test:` – when adding or updating tests
- `refactor:` – for code changes that do not fix a bug or add a feature
- `chore:` – for maintenance tasks, such as dependency updates or cleanup
- `style:` – for formatting changes, such as whitespace or commas
- `perf:` – for performance improvements
- `ci:` – for changes to CI/CD configurations or scripts
- `build:` – for changes to the build system or external dependencies
- `revert:` – for reverting a previous commit

Example:

```text
fix: handle nil pointer in route matcher
```

Each commit should be meaningful. When fixing bugs, include both the fix and the corresponding test where possible.

---

## 🧪 Testing

Make sure all tests pass before submitting a pull request.

Use Go’s built-in testing tools:

```bash
go test ./...
```

For concurrency-sensitive changes, also run:

```bash
go test -race ./...
```

If you add a new feature or fix a bug, write a test that verifies it where applicable.

---

## 🙋 Questions

If you’re unsure about anything, feel free to open an issue or ask in an existing one. We’re happy to help!

Thanks for contributing! 🚀
