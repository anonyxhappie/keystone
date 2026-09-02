# ForgeOS

> The adaptive engineering operating system for AI-assisted software development.

ForgeOS turns natural-language engineering requests into autonomous, evidence-backed software work while preserving project state, adapting execution to risk, and enforcing production-quality release gates.

ForgeOS is not limited to greenfield app generation. It is designed to handle new products, features, bug fixes, UI improvements, refactors, integrations, version work, reviews, QA, releases, continuation and recovery.

## Core principle

**The agent is disposable; project state is not.**

Models and agents may lose context, hit quotas, fail, or be replaced. ForgeOS keeps durable engineering state outside the model so work can resume safely.

## Design pillars

1. Persistent project state
2. Adaptive intent and workflow selection
3. Progressive, dependency-aware context compilation
4. Coherent engineering work slices
5. Risk-based validation without weakening release gates
6. Deterministic tooling and structured evidence
7. Checkpoint-based recovery
8. Requirement-to-release traceability
9. Consequence-based human escalation
10. Model/agent agnosticism

## Repository status

**Architecture/specification phase.** The system design is being established before implementation.

## Reference project

Losal is the first reference project for validating ForgeOS. ForgeOS core contracts remain project-agnostic.

See [`docs/FORGEOS_ARCHITECTURE_V1.md`](docs/FORGEOS_ARCHITECTURE_V1.md) for the initial architecture.