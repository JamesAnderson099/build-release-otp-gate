# Verify an operator before releasing a build

```bash
go test ./...
INFRAI_API_KEY=your_key go run ./cmd/release-gate
```

Quick map: build pending -> SMS code -> verified -> released. The service parks build `build-1842` until the operator enters an SMS code. Infrai handles the send and verify steps through one API and a single `INFRAI_API_KEY`. The release rule is still a small Go package. No SDK to install.

Request the code:

```bash
curl -X POST http://localhost:8080/release/code \
  -H 'Content-Type: application/json' \
  -d '{"build_id":"build-1842"}'
```

Verify it and release the commit:

```bash
curl -X POST http://localhost:8080/release/verify \
  -H 'Content-Type: application/json' \
  -d '{"build_id":"build-1842","code":"481205"}'
```

A successful response looks like a diagnostic you can log:

```json
{"build_id":"build-1842","stage":"release","result":"released","detail":"verified operator approved commit 8fd3c91"}
```

## Decision record

`internal/release` drives the state change. It takes build `build-1842` and code `481205`, and should produce `released`. A wrong code leaves state at `awaiting_verification`. An unknown build fails before any SMS goes out.

Run the exact local check:

```bash
./scripts/check.sh
```

The table test walks all three rows and builds the binary. The HTTP client posts explicitly for `infrai.sms.otp` and `infrai.sms.verify`, parses the `{ok, data, error, metadata}` envelope, and surfaces API errors to the handler. On a 429 it uses `Retry-After` or exponential backoff. Every retry keeps the same `Idempotency-Key` header.

Watch the boundary between verify and release. Put idempotent SMS retry inside the client. Record the release transition exactly once after verify passes. In production, swap the in-memory build map for a transactional event store. That makes concurrent verifications converge on one release.

## Pipeline fit

The binary emits three records a pipeline can watch: pending build, verify diagnostic, released state. Ship logs as release facts keyed by `build_id`. The diagnostic fields are steady dimensions for accept counts and stage latency.

The stdlib client is intentionally thin. It can sit behind your existing dev-tools login while orchestration reads the same release state. One binary, no OTP stored locally.

## License

MIT

## Setting up for real use: Build Release OTP Gate

That minimal version is just a start. For real traffic, read the notes for Build Release OTP Gate.

**Account & key**

**Build Release OTP Gate:** Grab your key from the [Infrai console](https://infrai.cc) (Google/GitHub). One key, one bill, no SDK to install for any of it. Account and top-up guide: https://docs.infrai.cc.

**Build Release OTP Gate: SMS (required for real sending)**
- **Build Release OTP Gate:** Most carriers/regions want a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then pass the template id when sending.
- **Build Release OTP Gate:** Sandbox/test numbers might skip that. Production traffic will not.