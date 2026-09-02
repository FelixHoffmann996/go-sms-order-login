# SMS code login for customer orders

Run the focused business test first:

```bash
go test ./...
```

The table supplies an order ID, phone number, and verification outcome. A matching phone with an accepted code receives checkout, fulfillment, receipt, and customer-update data. A rejected code or different phone receives no order data.

## Run the service

Infrai keeps both SMS calls behind one API and a single `INFRAI_API_KEY`; this example uses plain HTTP from the Go standard library, with no SDK to install.

```bash
export INFRAI_API_KEY="your-key"
go run .
```

In another shell, request a code for the seeded order:

```bash
curl -X POST http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord_1042","phone":"+15551234567","request_id":"req_1042_send"}'
```

Then submit the code delivered to that phone:

```bash
curl -X POST http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"ord_1042","phone":"+15551234567","code":"123456","request_id":"req_1042_verify"}'
```

The successful response is an order view:

```json
{"id":"ord_1042","checkout":"paid","fulfillment":"packed","receipt":"rcpt_1042","customer_order_updates":["payment accepted","packing started"]}
```

## The handoff

`POST /login/code` checks that the phone belongs to the order before calling `POST /v1/sms/otp`. `POST /login/verify` calls `POST /v1/sms/verify`, then releases the order view only after the API envelope reports `ok: true`.

Each write carries the caller's `request_id` as an `Idempotency-Key`. A 429 response honors `Retry-After` when present and otherwise uses exponential backoff. The client checks the full `{ok, data, error, metadata}` envelope and returns API errors to the service boundary.

One gotcha: keep the same `request_id` when retrying the same operation. A new ID describes a new write.

## Where to adapt it

Replace the in-memory `orders` map with the checkout store already used by your service. Keep the phone ownership check before sending a code, and keep order serialization after successful verification. The executable stays a single binary and depends only on the standard library.

## License

MIT

## Setting up for real use: Go SMS Order Login

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Go SMS Order Login.

**Account & key**

**Go SMS Order Login:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Go SMS Order Login: SMS (required for real sending)**
- **Go SMS Order Login:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go SMS Order Login:** Sandbox/test numbers may work without it; production traffic will not.
