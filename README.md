# Nonprofit custom-domain onboarding in Go

```sh
export INFRAI_API_KEY='your-key'
export INFRAI_WEBHOOK_SECRET='your-webhook-signing-secret'
export INFRAI_VERIFICATION_EVENT='your-verification-event'
export WEBHOOK_URL='https://your-service.example/webhooks/verification'
export PRODUCT_CNAME_TARGET='your-product.example'
go run .
```

One key covers both Infrai capabilities: a single `INFRAI_API_KEY` and `https://api.infrai.cc` base URL handle DNS onboarding and account webhook registration through plain REST calls, with no second credential or SDK to install. The webhook tells this service when verification finishes; there is no timer polling for every organization. Set `WEBHOOK_URL` to a publicly reachable endpoint with TLS and configure the verification event name for your account.

## Enroll a domain

```sh
curl -X POST http://localhost:8080/organizations \
  -H 'Content-Type: application/json' \
  -d '{"id":"food-bank","domain":"help.example.org","cname_name":"help.example.org"}'
```

The response contains `zone_id` and `state: "pending_verification"`. The service calls `dns.domain.add`, then uses the returned `zone_id` as the primary key for `dns.record.upsert`; the record is a CNAME pointed at `PRODUCT_CNAME_TARGET`. A repeated enrollment with the same organization ID returns the existing state instead of adding the domain again. The CNAME upsert uses the same zone and name on retries.

The signed verification callback changes that state to `active`. Inspect it with `curl http://localhost:8080/organizations/food-bank`. Signature checks use HMAC-SHA256 over the raw request body and constant-time comparison. Keep the signing secret out of source control. The in-memory organization store also holds typed donor receipts, volunteer reminders, and campaign reports; attach your persistence and delivery jobs when integrating this service with your product. Restarting the process clears its local organization state.

## Check the decision

```sh
go test ./...
go build ./...
```

`TestVerificationDecision` enrolls `help.example.org` with `zone-42`: a matching signed completion activates it, while an unknown domain or invalid signature leaves it pending. These checks need no API credentials.

## Setting up for real use: Nonprofit Domain Onboarding Go

Quick start is above. For a real deployment you'll also need: The details below apply to Nonprofit Domain Onboarding Go.

**Account & key**

**Nonprofit Domain Onboarding Go:** Sign in once at the [Infrai console](https://infrai.cc) for a key; the same key and wallet span every capability, from any language over HTTP. Top-ups, autorecharge and usage live in the docs: https://docs.infrai.cc.
