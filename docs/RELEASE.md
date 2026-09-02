# V2 release process

Keystone is packaged as an unversioned Go module (`github.com/anonyxhappie/keystone`),
allowing users to install and update the CLI directly via:

    go install github.com/anonyxhappie/keystone/cmd/keystone@latest

Release tags use semantic versioning matching the module path.

Run the release checks from the repository root:

    gofmt -w internal cmd/keystone
    GOCACHE=/tmp/keystone-go-build go test ./...
    GOCACHE=/tmp/keystone-go-build go vet ./...
    GOCACHE=/tmp/keystone-go-build go test -race ./...
    GOCACHE=/tmp/keystone-go-build go build -trimpath ./cmd/keystone
    git diff --check

Run CLI smoke checks in a disposable fixture:

    keystone init
    keystone doctor
    keystone status
    keystone ask "inspect the fixture"
    keystone run "inspect the fixture"
    keystone replay RUN-ID

Verify the real provider boundary on the release machine when a provider is in
scope:

    agy --version
    codex --version
    keystone doctor
    keystone run "make a small validated fixture change"

The provider smoke must use an authenticated CLI and must leave a durable
provider session id, normalized observations, deterministic validation, and a
verified completion. A protocol fixture test is not a substitute for this
smoke. If a provider is unavailable or unauthenticated, keep it marked PARTIAL
and do not publish a release claiming that provider is verified.

Do not tag while docs/IMPLEMENTATION_STATUS.json contains a material MISSING, BROKEN, or unresolved release-blocking PARTIAL capability. The release commit must be reviewed on the intended branch, and the annotated tag must point to that exact commit.
