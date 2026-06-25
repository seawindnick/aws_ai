#!/usr/bin/env bash
# End-to-end API verification against the deployed API Gateway endpoint.
# Usage:
#   ./ci/verify.sh                        # auto-detect endpoint from CloudFormation
#   API_URL=https://xxx.execute-api.us-east-1.amazonaws.com ./ci/verify.sh

set -euo pipefail

# ── Colour helpers ────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'; NC='\033[0m'
pass() { echo -e "${GREEN}[PASS]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; FAILURES=$((FAILURES + 1)); }
info() { echo -e "${YELLOW}[INFO]${NC} $*"; }

FAILURES=0

# ── Resolve API base URL ──────────────────────────────────────────────────────
if [[ -z "${API_URL:-}" ]]; then
  info "API_URL not set — querying CloudFormation stack WrongQuestion..."
  API_URL=$(aws cloudformation describe-stacks \
    --stack-name WrongQuestion \
    --region us-east-1 \
    --query "Stacks[0].Outputs[?contains(OutputKey,'ApiUrl') || contains(OutputKey,'HttpApi')].OutputValue | [0]" \
    --output text 2>/dev/null || true)
  if [[ -z "$API_URL" || "$API_URL" == "None" ]]; then
    echo "ERROR: Could not determine API_URL. Set it manually: API_URL=https://... ./ci/verify.sh"
    exit 1
  fi
fi
API_URL="${API_URL%/}"   # strip trailing slash
info "Testing against: $API_URL"

# ── Test helpers ──────────────────────────────────────────────────────────────
# http <method> <path> [curl-args...]  → stores body in $BODY, status in $STATUS
http() {
  local method="$1"; local path="$2"; shift 2
  local tmp; tmp=$(mktemp)
  STATUS=$(curl -s -o "$tmp" -w "%{http_code}" -X "$method" "$API_URL$path" "$@")
  BODY=$(cat "$tmp"); rm -f "$tmp"
}

assert_status() {
  local expected="$1" label="$2"
  if [[ "$STATUS" == "$expected" ]]; then
    pass "$label (HTTP $STATUS)"
  else
    fail "$label — expected HTTP $expected, got $STATUS. Body: $BODY"
  fi
}

assert_json_key() {
  local key="$1" label="$2"
  if echo "$BODY" | grep -q "\"$key\""; then
    pass "$label (found key '$key')"
  else
    fail "$label — key '$key' missing in: $BODY"
  fi
}

assert_no_key() {
  local key="$1" label="$2"
  if ! echo "$BODY" | grep -q "\"$key\""; then
    pass "$label (key '$key' absent as expected)"
  else
    fail "$label — key '$key' should NOT be present: $BODY"
  fi
}

# ── Test data ─────────────────────────────────────────────────────────────────
TS=$(date +%s)
USER_A_EMAIL="verify-a-${TS}@test.invalid"
USER_B_EMAIL="verify-b-${TS}@test.invalid"
PASSWORD="Verify@123!"
QUESTION_ID="q-${TS}-test"

echo ""
echo "════════════════════════════════════════"
echo " 1. Health check"
echo "════════════════════════════════════════"

http GET /api/health
assert_status 200 "GET /api/health"
assert_json_key "status" "health response has 'status' key"

echo ""
echo "════════════════════════════════════════"
echo " 2. Auth — unauthenticated rejection"
echo "════════════════════════════════════════"

http GET /api/questions
assert_status 401 "GET /api/questions without token → 401"

http GET /api/me
assert_status 401 "GET /api/me without token → 401"

http POST /api/questions \
  -H "Content-Type: application/json" \
  -d '{"question_id":"no-auth-test"}'
assert_status 401 "POST /api/questions without token → 401"

echo ""
echo "════════════════════════════════════════"
echo " 3. Signup"
echo "════════════════════════════════════════"

http POST /api/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_A_EMAIL\",\"password\":\"$PASSWORD\"}"
assert_status 201 "POST /api/auth/signup user-A"
assert_json_key "user_id" "signup response has 'user_id'"

http POST /api/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_B_EMAIL\",\"password\":\"$PASSWORD\"}"
assert_status 201 "POST /api/auth/signup user-B"

# Bad signup — missing password
http POST /api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"nopw@test.invalid"}'
assert_status 400 "POST /api/auth/signup missing password → 400"

# Duplicate email
http POST /api/auth/signup \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_A_EMAIL\",\"password\":\"$PASSWORD\"}"
assert_status 400 "POST /api/auth/signup duplicate email → 400"

echo ""
info "NOTE: Cognito email verification is required before login."
info "      Login and authenticated tests are skipped in CI (no real email)."
info "      To run the full suite, set TOKEN_A and TOKEN_B env vars:"
info "        TOKEN_A=<id_token_user_a> TOKEN_B=<id_token_user_b> ./ci/verify.sh"

if [[ -n "${TOKEN_A:-}" && -n "${TOKEN_B:-}" ]]; then
  echo ""
  echo "════════════════════════════════════════"
  echo " 4. Login"
  echo "════════════════════════════════════════"

  http POST /api/auth/login \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$USER_A_EMAIL\",\"password\":\"$PASSWORD\"}"
  assert_status 200 "POST /api/auth/login user-A"
  assert_json_key "access_token" "login returns access_token"
  assert_json_key "id_token"     "login returns id_token"

  http POST /api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"wrong@test.invalid","password":"WrongPw1!"}'
  assert_status 401 "POST /api/auth/login bad credentials → 401"

  echo ""
  echo "════════════════════════════════════════"
  echo " 5. User profile — GET /api/me"
  echo "════════════════════════════════════════"

  http GET /api/me -H "Authorization: Bearer $TOKEN_A"
  assert_status 200 "GET /api/me user-A"
  assert_json_key "user_id"   "me has user_id"
  assert_json_key "email"     "me has email"
  assert_json_key "role"      "me has role"
  assert_no_key   "password"  "me does not leak password field"

  echo ""
  echo "════════════════════════════════════════"
  echo " 6. Questions CRUD"
  echo "════════════════════════════════════════"

  # Create
  http POST /api/questions \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"question_id\":\"$QUESTION_ID\",\"subject\":\"math\"}"
  assert_status 201 "POST /api/questions create"
  assert_json_key "question_id" "create response has question_id"
  assert_json_key "status"      "create response has status"

  # Duplicate create → 409
  http POST /api/questions \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"question_id\":\"$QUESTION_ID\",\"subject\":\"math\"}"
  assert_status 409 "POST /api/questions duplicate → 409"

  # Missing question_id → 400
  http POST /api/questions \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d '{"subject":"math"}'
  assert_status 400 "POST /api/questions missing question_id → 400"

  # List
  http GET /api/questions -H "Authorization: Bearer $TOKEN_A"
  assert_status 200 "GET /api/questions list"
  assert_json_key "items" "list response has 'items' array"
  assert_json_key "count" "list response has 'count'"

  # Get by ID
  http GET "/api/questions/$QUESTION_ID" -H "Authorization: Bearer $TOKEN_A"
  assert_status 200 "GET /api/questions/:id owner"
  assert_json_key "question_id" "get response has question_id"

  echo ""
  echo "════════════════════════════════════════"
  echo " 7. User isolation (R1)"
  echo "════════════════════════════════════════"

  # user-B tries to read user-A's question → 403
  http GET "/api/questions/$QUESTION_ID" -H "Authorization: Bearer $TOKEN_B"
  assert_status 403 "GET /api/questions/:id cross-user → 403 (R1)"

  # user-B tries to delete user-A's question → 403
  http DELETE "/api/questions/$QUESTION_ID" -H "Authorization: Bearer $TOKEN_B"
  assert_status 403 "DELETE /api/questions/:id cross-user → 403 (R1)"

  # user-B list → should be empty (only sees own records)
  http GET /api/questions -H "Authorization: Bearer $TOKEN_B"
  assert_status 200 "GET /api/questions user-B sees only own records"
  B_COUNT=$(echo "$BODY" | grep -o '"count":[0-9]*' | grep -o '[0-9]*' || echo "0")
  if [[ "$B_COUNT" == "0" ]]; then
    pass "R1 isolation: user-B count=0 (no cross-user data)"
  else
    fail "R1 isolation: user-B got count=$B_COUNT, expected 0"
  fi

  echo ""
  echo "════════════════════════════════════════"
  echo " 8. Presigned upload URL"
  echo "════════════════════════════════════════"

  http POST /api/questions/upload-url \
    -H "Authorization: Bearer $TOKEN_A" \
    -H "Content-Type: application/json" \
    -d "{\"question_id\":\"$QUESTION_ID\",\"extension\":\"jpg\"}"
  assert_status 200 "POST /api/questions/upload-url"
  assert_json_key "upload_url"  "upload-url has 'upload_url'"
  assert_json_key "key"         "upload-url has 'key'"
  assert_json_key "question_id" "upload-url has 'question_id'"

  # Key must be scoped to user (R1 for S3 path)
  if echo "$BODY" | grep -q '"key":"images/'; then
    pass "upload key is under images/ prefix"
  else
    fail "upload key format unexpected: $BODY"
  fi

  echo ""
  echo "════════════════════════════════════════"
  echo " 9. 404 for unknown route"
  echo "════════════════════════════════════════"

  http GET /api/nonexistent -H "Authorization: Bearer $TOKEN_A"
  assert_status 404 "GET /api/nonexistent → 404"
  assert_json_key "error" "404 response has 'error' key"

  echo ""
  echo "════════════════════════════════════════"
  echo " 10. Cleanup — DELETE question"
  echo "════════════════════════════════════════"

  http DELETE "/api/questions/$QUESTION_ID" -H "Authorization: Bearer $TOKEN_A"
  assert_status 200 "DELETE /api/questions/:id owner"
  assert_json_key "deleted" "delete response has 'deleted' key"

  # Confirm deleted
  http GET "/api/questions/$QUESTION_ID" -H "Authorization: Bearer $TOKEN_A"
  assert_status 404 "GET deleted question → 404"
fi

echo ""
echo "════════════════════════════════════════"
echo " Results"
echo "════════════════════════════════════════"
if [[ "$FAILURES" -eq 0 ]]; then
  echo -e "${GREEN}All tests passed.${NC}"
  exit 0
else
  echo -e "${RED}$FAILURES test(s) failed.${NC}"
  exit 1
fi
