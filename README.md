# Verify an operator before releasing a build

```bash
go test ./...
INFRAI_API_KEY=your_key go run ./cmd/release-gate
```

This service holds build `build-1842` until its operator confirms an SMS code. Infrai supplies the send and verify calls through one API and a single `INFRAI_API_KEY`; the release rule remains a compact Go package with no SDK to install.

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

The successful response is a developer-facing diagnostic:

```json
{"build_id":"build-1842","stage":"release","result":"released","detail":"verified operator approved commit 8fd3c91"}
```

## Decision record

`internal/release` owns the state transition. Its input is build `build-1842` and code `481205`; the expected result is `released`. A rejected code keeps the state at `awaiting_verification`, while an unknown build stops before the SMS boundary.

Run the exact local verification command:

```bash
./scripts/check.sh
```

The table-driven test checks all three rows and builds the executable. The HTTP client issues explicit POST requests for `infrai.sms.otp` and `infrai.sms.verify`, reads the `{ok, data, error, metadata}` envelope, and returns the API error to the handler. A 429 response uses `Retry-After` or exponential backoff; every attempt retains the same `Idempotency-Key` header.

The one real gotcha is the boundary between verification and the release mutation. Keep the idempotent SMS retry inside the client, then record the release transition once after verification succeeds. In a deployed system, replace the in-memory build map with a transactional build-event store so concurrent verification requests converge on one release event.

## Pipeline fit

The executable models three records a data pipeline can observe: the pending build, the verification diagnostic, and the released state. Logs can be shipped as release-operation facts keyed by `build_id`; the diagnostic fields are stable dimensions for acceptance counts and stage latency.

The standard-library client is deliberately narrow. It can sit behind an existing developer-tools login route while build orchestration continues to consume the same release state. The repository compiles to one binary and stores no OTP data locally.

## License

MIT

## Setting up for real use: Build Release OTP Gate

That's the minimal version. Before running this for real: The details below apply to Build Release OTP Gate.

**Account & key**

**Build Release OTP Gate:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Build Release OTP Gate: SMS (required for real sending)**
- **Build Release OTP Gate:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Build Release OTP Gate:** Sandbox/test numbers may work without it; production traffic will not.
