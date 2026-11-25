package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/augustin-wien/augustina-backend/database"
	"github.com/augustin-wien/augustina-backend/ent"
	"github.com/augustin-wien/augustina-backend/mailer"
	"github.com/augustin-wien/augustina-backend/utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

var errSweepInjected = errors.New("injected failure")

// sweepDriver counts the statements, transaction starts and commits sent to the
// database and fails the one with number failAt
type sweepDriver struct {
	dialect.Driver
	calls  *atomic.Int64
	failAt int64
}

func (d *sweepDriver) hit() error {
	if d.calls.Add(1) == d.failAt {
		return errSweepInjected
	}
	return nil
}

func (d *sweepDriver) Exec(ctx context.Context, query string, args, v any) error {
	if err := d.hit(); err != nil {
		return err
	}
	return d.Driver.Exec(ctx, query, args, v)
}

func (d *sweepDriver) Query(ctx context.Context, query string, args, v any) error {
	if err := d.hit(); err != nil {
		return err
	}
	return d.Driver.Query(ctx, query, args, v)
}

func (d *sweepDriver) Tx(ctx context.Context) (dialect.Tx, error) { return d.BeginTx(ctx, nil) }

func (d *sweepDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	if err := d.hit(); err != nil {
		return nil, err
	}
	tx, err := d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	}).BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &sweepTx{Tx: tx, d: d}, nil
}

type sweepTx struct {
	dialect.Tx
	d *sweepDriver
}

func (t *sweepTx) Exec(ctx context.Context, query string, args, v any) error {
	if err := t.d.hit(); err != nil {
		return err
	}
	return t.Tx.Exec(ctx, query, args, v)
}

func (t *sweepTx) Query(ctx context.Context, query string, args, v any) error {
	if err := t.d.hit(); err != nil {
		return err
	}
	return t.Tx.Query(ctx, query, args, v)
}

func (t *sweepTx) Commit() error {
	if err := t.d.hit(); err != nil {
		_ = t.Tx.Rollback()
		return err
	}
	return t.Tx.Commit()
}

// sweepRequest is one request of a sweep
type sweepRequest struct {
	method, path, body, contentType string
}

// sweepHandler sends the request prepared by setup once for every database call
// it causes, failing that call. The handler must not panic and must answer, and
// once no call fails it must answer with want.
func sweepHandler(t *testing.T, want int, setup func(t *testing.T, run int) sweepRequest) {
	t.Helper()
	for failAt := int64(1); ; failAt++ {
		require.Less(t, failAt, int64(400), "too many database calls")
		req := setup(t, int(failAt))

		calls := &atomic.Int64{}
		orig := database.Db
		drv := &sweepDriver{Driver: entsql.OpenDB(dialect.Postgres, orig.DB), calls: calls, failAt: failAt}
		database.Db = database.Database{EntClient: ent.NewClient(ent.Driver(drv)), DB: orig.DB, IsProduction: orig.IsProduction}

		httpReq := httptest.NewRequest(req.method, req.path, strings.NewReader(req.body))
		httpReq.Header.Set("Authorization", "Bearer "+adminUserToken.AccessToken)
		if req.contentType != "" {
			httpReq.Header.Set("Content-Type", req.contentType)
		}
		var res *httptest.ResponseRecorder
		require.NotPanics(t, func() { res = utils.SubmitRequest(httpReq, r) }, "failing call %d", failAt)
		database.Db = orig

		require.NotZero(t, res.Code)
		if calls.Load() < failAt {
			require.Equal(t, want, res.Code, "without a failing call: %s", res.Body.String())
			return
		}
	}
}

var sweepSeq atomic.Int64

// sweepVendor creates a vendor with an account directly in the database
func sweepVendor(t *testing.T) (database.Vendor, database.Account) {
	t.Helper()
	n := sweepSeq.Add(1)
	id, err := database.Db.CreateVendor(database.Vendor{
		FirstName: "Sweep", LastName: "Vendor", Email: fmt.Sprintf("sweep-%d@vendor.test", n),
		LicenseID: null.StringFrom(fmt.Sprintf("sweep-%d", n)),
	})
	require.NoError(t, err)
	v, err := database.Db.GetVendor(id)
	require.NoError(t, err)
	a, err := database.Db.GetAccountByVendorID(id)
	require.NoError(t, err)
	return v, a
}

func sweepItem(t *testing.T, kind string) int {
	t.Helper()
	id, err := database.Db.CreateItem(database.Item{Name: fmt.Sprintf("Sweep item %d", sweepSeq.Add(1)), Description: "Test item", Price: 300, Type: kind})
	require.NoError(t, err)
	return id
}

// sweepVerifiedOrder creates a verified order of item sold by the vendor
func sweepVerifiedOrder(t *testing.T, v database.Vendor, a database.Account, item int) int {
	t.Helper()
	buyer, err := database.Db.GetAccountTypeID("UserAnon")
	require.NoError(t, err)
	id, err := database.Db.CreateOrder(database.Order{
		Vendor: v.ID, OrderCode: null.StringFrom(fmt.Sprintf("%d", 8100000000+sweepSeq.Add(1))),
		CustomerEmail: null.StringFrom(fmt.Sprintf("buyer-%d@example.com", sweepSeq.Add(1))),
		Entries:       []database.OrderEntry{{Item: item, Quantity: 1, Sender: buyer, Receiver: a.ID, IsSale: true}},
	})
	require.NoError(t, err)
	require.NoError(t, database.Db.VerifyOrderAndCreatePayments(id, 0))
	return id
}

func TestHandlersSurviveEachDatabaseFailure(t *testing.T) {
	mutex_test.Lock()
	defer mutex_test.Unlock()
	require.NoError(t, database.Db.InitEmptyTestDb())

	origSend := mailer.Send
	t.Cleanup(func() { mailer.Send = origSend })
	mailer.Send = func(*mailer.EmailRequest) (bool, error) { return true, nil }

	month := time.Now().Format("2006-01")
	get := func(path string) func(*testing.T, int) sweepRequest {
		return func(*testing.T, int) sweepRequest { return sweepRequest{method: "GET", path: path} }
	}

	cases := []struct {
		name  string
		want  int
		setup func(t *testing.T, run int) sweepRequest
	}{
		{"vendor with balance", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			sweepVerifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			return sweepRequest{method: "GET", path: fmt.Sprintf("/api/vendors/%d/", v.ID)}
		}},
		{"payments for payout", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			sweepVerifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			return sweepRequest{method: "GET", path: "/api/payments/forpayout/?vendor=" + v.LicenseID.String}
		}},
		{"payout", 200, func(t *testing.T, _ int) sweepRequest {
			v, a := sweepVendor(t)
			sweepVerifiedOrder(t, v, a, sweepItem(t, "normal_item"))
			return sweepRequest{method: "POST", path: "/api/payments/payout/", body: `{"VendorLicenseID":"` + v.LicenseID.String + `"}`}
		}},
		{"pos order", 200, func(t *testing.T, _ int) sweepRequest {
			v, _ := sweepVendor(t)
			return sweepRequest{method: "POST", path: "/api/vendors/" + v.LicenseID.String + "/pos-order/", body: fmt.Sprintf(`{"entries":[{"item":%d,"quantity":1}]}`, sweepItem(t, "normal_item"))}
		}},
		{"delete item with license", 204, func(t *testing.T, _ int) sweepRequest {
			return sweepRequest{method: "DELETE", path: fmt.Sprintf("/api/items/%d/?deleteLicense=true", sweepItem(t, "normal_item"))}
		}},
		{"restore item", 204, func(t *testing.T, _ int) sweepRequest {
			id := sweepItem(t, "normal_item")
			require.NoError(t, database.Db.DeleteItem(id))
			return sweepRequest{method: "POST", path: fmt.Sprintf("/api/items/%d/restore/", id)}
		}},
		{"create customer", 201, func(t *testing.T, _ int) sweepRequest {
			return sweepRequest{method: "POST", path: "/api/customers/", body: fmt.Sprintf(`{"email":"sweep-customer-%d@example.com"}`, sweepSeq.Add(1))}
		}},
		{"create abonement", 201, func(t *testing.T, _ int) sweepRequest {
			c, err := database.Db.CreateCustomer(&database.Customer{Email: fmt.Sprintf("sweep-abo-%d@example.com", sweepSeq.Add(1))})
			require.NoError(t, err)
			return sweepRequest{method: "POST", path: "/api/abonements/", body: fmt.Sprintf(`{"customer_id":%d,"from_date":"2026-01-01","to_date":"2026-12-31","status":"active"}`, c.ID)}
		}},
		{"create campaign", 200, func(t *testing.T, _ int) sweepRequest {
			return sweepRequest{method: "POST", path: "/api/campaigns/", body: fmt.Sprintf(`{"name":"sweep-%d","item_id":%d,"title":"t","enabled":true}`, sweepSeq.Add(1), sweepItem(t, "normal_item"))}
		}},
		{"create location", 200, func(t *testing.T, _ int) sweepRequest {
			v, _ := sweepVendor(t)
			return sweepRequest{method: "POST", path: fmt.Sprintf("/api/vendors/%d/locations/", v.ID), body: `{"name":"Billa","address":"Street 1","zip":"1010"}`}
		}},
		{"create comment", 200, func(t *testing.T, _ int) sweepRequest {
			v, _ := sweepVendor(t)
			return sweepRequest{method: "POST", path: fmt.Sprintf("/api/vendors/%d/comments/", v.ID), body: `{"comment":"x"}`}
		}},
		{"check license id", 200, func(t *testing.T, _ int) sweepRequest {
			v, _ := sweepVendor(t)
			return sweepRequest{method: "GET", path: "/api/vendors/check/" + v.LicenseID.String + "/"}
		}},
		{"update settings", 200, func(t *testing.T, _ int) sweepRequest {
			body := "--b\r\nContent-Disposition: form-data; name=\"MaxOrderAmount\"\r\n\r\n5000\r\n--b--\r\n"
			return sweepRequest{method: "PUT", path: "/api/settings/", body: body, contentType: "multipart/form-data; boundary=b"}
		}},
		{"shop items", 200, get("/api/items/")},
		{"backoffice items", 200, get("/api/items/backoffice/?includeArchived=true")},
		{"vendors", 200, get("/api/vendors/")},
		{"vendor statistics", 200, get("/api/vendors/statistics/")},
		{"payment statistics", 200, get("/api/payments/statistics/")},
		{"order statistics", 200, get("/api/orders/statistics/")},
		{"customers", 200, get("/api/customers/")},
		{"active abonements", 200, get("/api/abonements/active/")},
		{"locations", 200, get("/api/locations/")},
		{"map", 200, get("/api/map/")},
		{"monthly report", 200, get("/api/reports/monthly/" + month + "/")},
		{"monthly report csv", 200, get("/api/reports/monthly/" + month + "/csv/")},
		{"external payments", 200, get("/api/external-payments/")},
		{"public settings", 200, get("/api/settings/")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sweepHandler(t, tc.want, tc.setup)
		})
	}
}
