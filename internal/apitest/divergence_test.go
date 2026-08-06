package apitest

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// The behaviors pinned here are the ones the subscriptions service implements
// differently. Responses alone don't capture them, so these assert database
// state. When QMS is merged into subscriptions, exactly one implementation of
// each survives, and these tests say which one that has to be.

// queryString runs a single-string query against the test database.
func queryString(t *testing.T, query string, args ...any) string {
	t.Helper()
	var value string
	if err := testDB.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("query failed: %s\nquery: %s", err, query)
	}
	return value
}

// queryInt runs a single-integer query against the test database.
func queryInt(t *testing.T, query string, args ...any) int {
	t.Helper()
	var value int
	if err := testDB.QueryRow(query, args...).Scan(&value); err != nil {
		t.Fatalf("query failed: %s\nquery: %s", err, query)
	}
	return value
}

// recordUsage posts a usage update and fails the test if it isn't accepted.
func recordUsage(t *testing.T, username, resource, updateType string, value float64) {
	t.Helper()
	body := fmt.Sprintf(
		`{"username": %q, "resource_name": %q, "usage_value": %v, "update_type": %q, "metadata": "{}"}`,
		username, resource, value, updateType,
	)
	got := do(t, http.MethodPost, "/v1/usages", body)
	if got.status != http.StatusOK {
		t.Fatalf("recording usage failed: status %d, body %s", got.status, got.body)
	}
}

// QMS writes an audit row to `updates` for every usage change, alongside the
// running total in `usages`. The subscriptions service computes the total in
// SQL and writes no `updates` row on its usage path, so a merge that keeps the
// wrong implementation loses the audit trail without any visible API change.
func TestUsageWritesAnUpdatesRow(t *testing.T) {
	resetDB(t)
	createUser(t, testUser)

	recordUsage(t, testUser, "cpu.hours", "SET", 10)
	recordUsage(t, testUser, "cpu.hours", "ADD", 5)

	updateCount := queryInt(t, `
		SELECT count(*)
		  FROM updates u
		  JOIN users ON u.user_id = users.id
		 WHERE users.username = $1`, "testuser")
	if updateCount != 2 {
		t.Errorf("updates rows = %d, want 2 (one per usage change)", updateCount)
	}

	// KNOWN BUG, recorded deliberately: both rows say ADD even though the
	// first update was a SET. The operation is looked up with
	//
	//	updateOperation := model.UpdateOperation{Name: usage.UpdateType}
	//	tx.First(&updateOperation)
	//
	// but GORM's First only filters on the primary key, ignoring Name, so it
	// always returns whichever row sorts first — ADD. The running total in
	// `usages` is still correct because the SET/ADD arithmetic is a separate
	// switch; it is only the audit trail that misreports. The same unreachable
	// lookup is why an invalid update type 500s instead of 400ing.
	operations := queryString(t, `
		SELECT string_agg(o.name, ',' ORDER BY u.effective_date)
		  FROM updates u
		  JOIN users ON u.user_id = users.id
		  JOIN update_operations o ON u.update_operation_id = o.id
		 WHERE users.username = $1`, "testuser")
	if operations != "ADD,ADD" {
		t.Errorf("recorded operations = %q, want \"ADD,ADD\" (the current, buggy behavior)", operations)
	}

	usage := queryString(t, `
		SELECT us.usage::text
		  FROM usages us
		  JOIN subscriptions s ON us.subscription_id = s.id
		  JOIN users ON s.user_id = users.id
		 WHERE users.username = $1`, "testuser")
	if usage != "15" {
		t.Errorf("running usage total = %s, want 15", usage)
	}
}

// KNOWN BUG, recorded deliberately: POST /v1/subscriptions?force=false fails
// for every request, and reports success while doing it.
//
// The unforced path is supposed to compare the plans' default cpu.hours
// allocations and re-subscribe only on an upgrade. It never gets that far,
// because GetActiveSubscriptionForDate builds
//
//	.Or("? > subscriptions.effective_start_date AND ...")
//
// with no argument for the placeholder, so PostgreSQL rejects the statement
// with `syntax error at or near ">"`. The handler turns that into a per-item
// failure_reason and still answers 200, so terrain's
// POST /terrain/admin/qms/subscriptions silently creates nothing whenever
// force isn't set.
//
// This test pins the broken behavior only so the merge can't quietly alter it.
// When the placeholder is fixed, replace it with the upgrade/refuse matrix the
// rule is meant to implement — the plan allocations it depends on are already
// correct (Basic 200, Pro 20000), and model.TestGetDefaultQuotaValue covers the
// selector that reads them.
func TestUnforcedSubscriptionChangeIsBroken(t *testing.T) {
	resetDB(t)

	forced := fmt.Sprintf(
		`{"subscriptions": [{"username": %q, "plan_name": "Basic", "paid": true, "periods": 1}]}`,
		testUser,
	)
	if got := do(t, http.MethodPost, "/v1/subscriptions?force=true", forced); got.status != http.StatusOK {
		t.Fatalf("unable to establish the starting subscription: %s", got.body)
	}

	body := fmt.Sprintf(
		`{"subscriptions": [{"username": %q, "plan_name": "Pro", "paid": true, "periods": 1}]}`,
		testUser,
	)
	got := do(t, http.MethodPost, "/v1/subscriptions?force=false", body)
	if got.status != http.StatusOK {
		t.Fatalf("status = %d, body %s", got.status, got.body)
	}

	decoded := mustDecode(t, got)
	results, ok := decoded["result"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected exactly one subscription result, got: %s", got.body)
	}
	result := results[0].(map[string]any)

	reason, _ := result["failure_reason"].(string)
	if !strings.Contains(reason, "syntax error") {
		t.Errorf("failure_reason = %q, want the SQL syntax error this path currently produces", reason)
	}
	if newSubscription, _ := result["new_subscription"].(bool); newSubscription {
		t.Error("new_subscription = true, want false: the request failed")
	}

	// The starting subscription is untouched, so the upgrade never happened.
	activePlan := queryString(t, `
		SELECT p.name
		  FROM subscriptions s
		  JOIN users ON s.user_id = users.id
		  JOIN plans p ON s.plan_id = p.id
		 WHERE users.username = $1
		   AND CURRENT_TIMESTAMP BETWEEN s.effective_start_date AND s.effective_end_date`,
		testUser)
	if activePlan != "Basic" {
		t.Errorf("active plan = %q, want \"Basic\"", activePlan)
	}
}

// Creating an overlapping subscription deactivates the previous one, so a user
// never has two active subscriptions at once. Anything reading "the current
// plan" depends on that invariant holding.
func TestOverlappingSubscriptionsAreDeactivated(t *testing.T) {
	resetDB(t)

	for _, plan := range []string{"Basic", "Pro", "Commercial"} {
		body := fmt.Sprintf(
			`{"subscriptions": [{"username": %q, "plan_name": %q, "paid": true, "periods": 1}]}`,
			testUser, plan,
		)
		if got := do(t, http.MethodPost, "/v1/subscriptions?force=true", body); got.status != http.StatusOK {
			t.Fatalf("unable to subscribe to %s: %s", plan, got.body)
		}
	}

	activeCount := queryInt(t, `
		SELECT count(*)
		  FROM subscriptions s
		  JOIN users ON s.user_id = users.id
		 WHERE users.username = $1
		   AND CURRENT_TIMESTAMP BETWEEN s.effective_start_date AND s.effective_end_date`,
		testUser)
	if activeCount != 1 {
		t.Errorf("active subscriptions = %d, want exactly 1", activeCount)
	}

	totalCount := queryInt(t, `
		SELECT count(*)
		  FROM subscriptions s
		  JOIN users ON s.user_id = users.id
		 WHERE users.username = $1`, testUser)
	if totalCount != 3 {
		t.Errorf("total subscriptions = %d, want 3 (the history is retained)", totalCount)
	}
}

// KNOWN BUG, recorded deliberately: the two ways of subscribing a user store
// two different usernames for the same person. PUT /v1/users/{username} and the
// usage endpoints trim the configured suffix, but POST /v1/subscriptions passes
// req.Username through untouched, so an admin bulk-subscribing "user@domain"
// creates a second user row that the user's own /terrain/qms/user/plan — which
// looks up the trimmed name — never sees.
//
// Terrain reaches both paths: POST /terrain/admin/qms/subscriptions is the bulk
// endpoint, while /terrain/qms/user/* are the trimming ones. Fix this before or
// during the merge, and expect existing databases to hold duplicate user rows
// that need reconciling.
func TestBulkSubscriptionsDoNotTrimTheUsernameSuffix(t *testing.T) {
	resetDB(t)

	body := fmt.Sprintf(
		`{"subscriptions": [{"username": %q, "plan_name": "Pro", "paid": true, "periods": 1}]}`,
		testUser,
	)
	if got := do(t, http.MethodPost, "/v1/subscriptions?force=true", body); got.status != http.StatusOK {
		t.Fatalf("unable to create the subscription: %s", got.body)
	}

	// The bulk endpoint keeps the suffix.
	bulkCount := queryInt(t, `SELECT count(*) FROM users WHERE username = $1`, testUser)
	if bulkCount != 1 {
		t.Errorf("users rows named %q = %d, want 1", testUser, bulkCount)
	}

	// The user-scoped endpoint trims it, producing a second row for the same person.
	createUser(t, testUser)
	trimmedCount := queryInt(t, `SELECT count(*) FROM users WHERE username = $1`, "testuser")
	if trimmedCount != 1 {
		t.Errorf("users rows named %q = %d, want 1", "testuser", trimmedCount)
	}

	total := queryInt(t, `SELECT count(*) FROM users`)
	if total != 2 {
		t.Errorf("total users = %d, want 2 (one per endpoint, for the same person)", total)
	}
}

// QMS resolves the resource type by name against whatever subscription is
// currently active. The subscriptions service requires the caller to supply
// both the subscription and resource-type UUIDs, so terrain's admin quota
// route only works against this implementation.
func TestQuotaUpdateResolvesResourceTypeByName(t *testing.T) {
	resetDB(t)
	createUser(t, testUser)

	path := fmt.Sprintf("/v1/users/%s/plan/cpu.hours/quota", testUser)
	if got := do(t, http.MethodPost, path, `{"quota": 750}`); got.status != http.StatusOK {
		t.Fatalf("unable to set the quota: %s", got.body)
	}

	quota := queryString(t, `
		SELECT q.quota::text
		  FROM quotas q
		  JOIN subscriptions s ON q.subscription_id = s.id
		  JOIN users ON s.user_id = users.id
		  JOIN resource_types rt ON q.resource_type_id = rt.id
		 WHERE users.username = $1
		   AND rt.name = 'cpu.hours'`, "testuser")
	if quota != "750" {
		t.Errorf("cpu.hours quota = %s, want 750", quota)
	}
}
