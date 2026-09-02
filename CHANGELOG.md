# Changelog

## 2.0.1

Correct the Go module path and package installation path for the V2 major release. The module is now published as `github.com/anonyxhappie/keystone/v2`, with matching internal imports and installation documentation.

## 2.0.0

The V2 control-plane implementation adds the executable canonical lifecycle, durable event journal and snapshot recovery, local-process and manual harness boundaries, normalized observations, evidence-backed verification, deterministic supervision, bounded full-auto execution, checkpoints, pause/approval/stop controls, learning lifecycle, harness switching, project intelligence, delivery state, policy enforcement, artifact redaction, replay, and the complete CLI.

Known limitations are recorded in docs/IMPLEMENTATION_STATUS.md and docs/IMPLEMENTATION_STATUS.json. Remote CI/browser/operations execution and semantic model review remain explicit adapter boundaries.
