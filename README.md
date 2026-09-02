# Keystone

> Persistent engineering intelligence between human intent and AI development harnesses.

Keystone is an open-source, language-, framework-, architecture-, and harness-agnostic engineering intelligence layer. It sits between a user's request and existing AI development harnesses such as Codex, Antigravity, Claude Code, and future tools.

Keystone does not replace the harness. It maintains durable project state, compiles relevant context, generates harness-specific instructions, observes results, evaluates correctness and agent behavior, detects hallucination, instruction drift, loops, unnecessary work, excessive token usage, architectural inconsistency, and weak methodology, and can continue autonomously within explicit policies.

The long-term goal is a persistent engineering control layer spanning development, validation, delivery, and production operations.

## Design principles

- **Harness-agnostic:** existing coding agents remain the execution layer.
- **Project-owned state:** engineering context survives model, agent, machine, and context changes.
- **Evidence over claims:** completion is verified through observable artifacts and deterministic checks.
- **Autonomy within policy:** full-auto operation is possible without unrestricted destructive authority.
- **Adaptive supervision:** Keystone evaluates not only outcomes, but how effectively the harness worked.
- **Project-specific learning:** repeated successes and mistakes improve future context, strategy, and supervision.
- **Language/framework agnostic:** the core model does not assume a programming language or application architecture.
- **Extensible capabilities:** testing, browser validation, Git, CI/CD, observability, monitoring, alerting, and incident workflows can be added without changing the core model.

## Repository

This repository is the Keystone project itself. Architecture and implementation specifications will be added as the design is hardened before the first production implementation.
