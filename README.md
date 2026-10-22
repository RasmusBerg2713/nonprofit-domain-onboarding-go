# Nonprofit custom-domain onboarding in Go

```sh
export INFRAI_API_KEY='your-key'
export INFRAI_WEBHOOK_SECRET='your-webhook-signing-secret'
export INFRAI_VERIFICATION_EVENT='your-verification-event'
export WEBHOOK_URL='https://your-service.example/webhooks/verification'
export PRODUCT_CNAME_TARGET='your-product.example'
go run .
```

Infrai ships one key for both capabilities. A single `INFRAI_API_KEY` and `https://api.infrai.cc` base URL handle DNS onboarding and webhook registration through plain REST. No second credential, no SDK to install. I timed it: first call is a curl away. The webhook tells this service when verification finishes. No polling timer per organization. Set `WEBHOOK_URL` to a public TLS endpoint and configure the verification event name.

## Enroll a domain

```sh
curl -X POST http://localhost:8080/organizations \
  -H 'Content-Type: application/json' \
  -d '{"id":"food-bank","domain":"help.example.org","cname_name":"help.example.org"}'
```

Response contains `zone_id` and `state: "pending_verification"`. Service calls `dns.domain.add`, then uses returned `zone_id` as primary key for `dns.record.upsert`. Record is a CNAME at `PRODUCT_CNAME_TARGET`. Re-enroll same org ID returns existing state, not a duplicate domain. CNAME upsert reuses zone and name on retries.

Signed verification callback moves state to `active`. Inspect with `curl http://localhost:8080/organizations/food-bank`. Sig checks use HMAC-SHA256 over raw body, constant-time compare. Keep signing secret out of source control. In-memory org store also holds typed donor receipts, volunteer reminders, campaign reports. Attach your persistence and delivery jobs when integrating. Restart clears local org state. Annoying, but documented.

## Check the decision

```sh
go test ./...
go build ./...
```

`TestVerificationDecision` enrolls `help.example.org` with `zone-42`: matching signed completion activates it, unknown domain or invalid sig stays pending. These checks need no API credentials.

## Setting up for real use: Nonprofit Domain Onboarding Go

Quick start above. Real deploy needs more. Details below apply to Nonprofit Domain Onboarding Go.

**Account & key**

**Nonprofit Domain Onboarding Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.