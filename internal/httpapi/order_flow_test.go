package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRootRedirectsToMenu(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("GET / status = %d, want %d", resp.Code, http.StatusSeeOther)
	}

	if location := resp.Header().Get("Location"); location != "/menu" {
		t.Errorf("Location = %q, want /menu", location)
	}
}

func TestCartAddUpdateRemoveFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	// 1. Add Americano Ice to cart
	form := url.Values{}
	form.Set("product_slug", "americano")
	form.Set("variant", "Ice")

	req := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("POST /cart/add status = %d, want %d", resp.Code, http.StatusSeeOther)
	}

	cookie := resp.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "pds_cart=") {
		t.Fatalf("Set-Cookie missing pds_cart: %q", cookie)
	}

	// Extract raw cookie value
	rawCookie := strings.Split(strings.Split(cookie, ";")[0], "=")[1]

	// 2. View cart page with cookie
	reqCart := httptest.NewRequest(http.MethodGet, "/keranjang", nil)
	reqCart.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookie})
	respCart := httptest.NewRecorder()
	router.ServeHTTP(respCart, reqCart)

	if respCart.Code != http.StatusOK {
		t.Fatalf("GET /keranjang status = %d, want %d", respCart.Code, http.StatusOK)
	}

	body := respCart.Body.String()
	if !strings.Contains(body, "Americano") || !strings.Contains(body, "Ice") || !strings.Contains(body, "Rp 15.000") {
		t.Errorf("cart page does not contain item details: %s", body)
	}

	// 3. Update qty (+1 -> qty=2)
	formUpdate := url.Values{}
	formUpdate.Set("index", "0")
	formUpdate.Set("delta", "1")

	reqUpdate := httptest.NewRequest(http.MethodPost, "/cart/update", strings.NewReader(formUpdate.Encode()))
	reqUpdate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqUpdate.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookie})
	respUpdate := httptest.NewRecorder()
	router.ServeHTTP(respUpdate, reqUpdate)

	if respUpdate.Code != http.StatusSeeOther {
		t.Fatalf("POST /cart/update status = %d, want %d", respUpdate.Code, http.StatusSeeOther)
	}

	cookieUpdated := respUpdate.Header().Get("Set-Cookie")
	rawCookieUpdated := strings.Split(strings.Split(cookieUpdated, ";")[0], "=")[1]

	// Check cart page with updated cookie
	reqCart2 := httptest.NewRequest(http.MethodGet, "/keranjang", nil)
	reqCart2.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookieUpdated})
	respCart2 := httptest.NewRecorder()
	router.ServeHTTP(respCart2, reqCart2)

	body2 := respCart2.Body.String()
	if !strings.Contains(body2, "Rp 30.000") {
		t.Errorf("expected total Rp 30.000 for 2 Americano Ice, got: %s", body2)
	}

	// 4. Remove item
	formRemove := url.Values{}
	formRemove.Set("index", "0")
	reqRemove := httptest.NewRequest(http.MethodPost, "/cart/remove", strings.NewReader(formRemove.Encode()))
	reqRemove.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqRemove.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookieUpdated})
	respRemove := httptest.NewRecorder()
	router.ServeHTTP(respRemove, reqRemove)

	if respRemove.Code != http.StatusSeeOther {
		t.Fatalf("POST /cart/remove status = %d, want %d", respRemove.Code, http.StatusSeeOther)
	}
}

func TestCheckoutValidationAndSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	// 1. Add item to get a valid cart cookie
	formAdd := url.Values{}
	formAdd.Set("product_slug", "espresso")
	reqAdd := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(formAdd.Encode()))
	reqAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respAdd := httptest.NewRecorder()
	router.ServeHTTP(respAdd, reqAdd)

	cookie := respAdd.Header().Get("Set-Cookie")
	rawCookie := strings.Split(strings.Split(cookie, ";")[0], "=")[1]

	// 2. Checkout with invalid data (dine_in missing table number)
	formInvalid := url.Values{}
	formInvalid.Set("customer_name", "Ahmad")
	formInvalid.Set("customer_phone", "08123456789")
	formInvalid.Set("order_type", "dine_in")
	formInvalid.Set("table_number", "")
	formInvalid.Set("payment_method", "cash")

	reqCheckoutFail := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(formInvalid.Encode()))
	reqCheckoutFail.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCheckoutFail.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookie})
	respCheckoutFail := httptest.NewRecorder()
	router.ServeHTTP(respCheckoutFail, reqCheckoutFail)

	if respCheckoutFail.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 Unprocessable Entity", respCheckoutFail.Code)
	}
	if !strings.Contains(respCheckoutFail.Body.String(), "nomor meja wajib diisi") {
		t.Errorf("error message missing in body: %s", respCheckoutFail.Body.String())
	}

	// 3. Checkout with valid data
	formValid := url.Values{}
	formValid.Set("customer_name", "Ahmad Fauzi")
	formValid.Set("customer_phone", "081234567890")
	formValid.Set("order_type", "dine_in")
	formValid.Set("table_number", "07")
	formValid.Set("notes", "Tanpa gula")
	formValid.Set("payment_method", "cash")

	reqCheckoutSuccess := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(formValid.Encode()))
	reqCheckoutSuccess.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCheckoutSuccess.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookie})
	respCheckoutSuccess := httptest.NewRecorder()
	router.ServeHTTP(respCheckoutSuccess, reqCheckoutSuccess)

	if respCheckoutSuccess.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303 See Other", respCheckoutSuccess.Code)
	}

	location := respCheckoutSuccess.Header().Get("Location")
	if !strings.HasPrefix(location, "/pesanan/PDS-") {
		t.Fatalf("redirect location %q should start with /pesanan/PDS-", location)
	}

	orderCode := strings.TrimPrefix(location, "/pesanan/")

	// 4. View order status page
	reqOrder := httptest.NewRequest(http.MethodGet, location, nil)
	respOrder := httptest.NewRecorder()
	router.ServeHTTP(respOrder, reqOrder)

	if respOrder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", location, respOrder.Code)
	}

	orderBody := respOrder.Body.String()
	if !strings.Contains(orderBody, orderCode) {
		t.Errorf("receipt missing order code: %s", orderBody)
	}
	if !strings.Contains(orderBody, "Ahmad Fauzi") || !strings.Contains(orderBody, "Espresso") {
		t.Errorf("receipt missing customer or product info: %s", orderBody)
	}

	// 5. Track page lookup
	formTrack := url.Values{}
	formTrack.Set("code", orderCode)
	reqTrack := httptest.NewRequest(http.MethodPost, "/lacak", strings.NewReader(formTrack.Encode()))
	reqTrack.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respTrack := httptest.NewRecorder()
	router.ServeHTTP(respTrack, reqTrack)

	if respTrack.Code != http.StatusSeeOther {
		t.Fatalf("POST /lacak status = %d, want 303", respTrack.Code)
	}
	if trackLocation := respTrack.Header().Get("Location"); trackLocation != location {
		t.Errorf("track redirected to %q, want %q", trackLocation, location)
	}

	// 6. Customer can cancel pending order
	reqCancel := httptest.NewRequest(http.MethodPost, "/pesanan/"+orderCode+"/batal", nil)
	respCancel := httptest.NewRecorder()
	router.ServeHTTP(respCancel, reqCancel)

	if respCancel.Code != http.StatusSeeOther {
		t.Fatalf("POST /pesanan/%s/batal status = %d, want 303", orderCode, respCancel.Code)
	}

	// Verify order status is Dibatalkan
	reqOrderCancelled := httptest.NewRequest(http.MethodGet, location, nil)
	respOrderCancelled := httptest.NewRecorder()
	router.ServeHTTP(respOrderCancelled, reqOrderCancelled)
	if !strings.Contains(respOrderCancelled.Body.String(), "Dibatalkan") {
		t.Errorf("cancelled order should show Dibatalkan status")
	}
}

func TestQRISPaymentAndWebhookFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	// 1. Add item to cart
	formAdd := url.Values{}
	formAdd.Set("product_slug", "caramel")
	formAdd.Set("variant", "Ice")

	reqAdd := httptest.NewRequest(http.MethodPost, "/cart/add", strings.NewReader(formAdd.Encode()))
	reqAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respAdd := httptest.NewRecorder()
	router.ServeHTTP(respAdd, reqAdd)

	cookie := respAdd.Header().Get("Set-Cookie")
	rawCookie := strings.Split(strings.Split(cookie, ";")[0], "=")[1]

	// 2. Checkout with QRIS
	formCheckout := url.Values{}
	formCheckout.Set("customer_name", "Bambang")
	formCheckout.Set("customer_phone", "081987654321")
	formCheckout.Set("order_type", "takeaway")
	formCheckout.Set("payment_method", "qris")

	reqCheckout := httptest.NewRequest(http.MethodPost, "/checkout", strings.NewReader(formCheckout.Encode()))
	reqCheckout.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCheckout.AddCookie(&http.Cookie{Name: "pds_cart", Value: rawCookie})
	respCheckout := httptest.NewRecorder()
	router.ServeHTTP(respCheckout, reqCheckout)

	if respCheckout.Code != http.StatusSeeOther {
		t.Fatalf("checkout status = %d, want 303", respCheckout.Code)
	}

	location := respCheckout.Header().Get("Location")
	orderCode := strings.TrimPrefix(location, "/pesanan/")

	// 3. View receipt - must contain QRIS card and QR image
	reqOrder := httptest.NewRequest(http.MethodGet, location, nil)
	respOrder := httptest.NewRecorder()
	router.ServeHTTP(respOrder, reqOrder)

	receiptBody := respOrder.Body.String()
	if !strings.Contains(receiptBody, "qris-payment-card") {
		t.Errorf("receipt should contain qris-payment-card")
	}
	if !strings.Contains(receiptBody, "QRIS Dinamis Otomatis") {
		t.Errorf("receipt should indicate QRIS Dinamis Otomatis")
	}

	// 4. Check status API before payment
	reqStatus := httptest.NewRequest(http.MethodGet, "/api/pesanan/"+orderCode+"/status", nil)
	respStatus := httptest.NewRecorder()
	router.ServeHTTP(respStatus, reqStatus)

	if !strings.Contains(respStatus.Body.String(), `"is_paid":false`) {
		t.Errorf("order status before payment should be is_paid: false, got: %s", respStatus.Body.String())
	}

	// 5. Send Midtrans Webhook (Settlement)
	webhookPayload := fmt.Sprintf(`{
		"order_id": %q,
		"status_code": "200",
		"gross_amount": "18000.00",
		"transaction_status": "settlement",
		"payment_type": "qris"
	}`, orderCode)

	reqWebhook := httptest.NewRequest(http.MethodPost, "/webhook/midtrans", strings.NewReader(webhookPayload))
	reqWebhook.Header.Set("Content-Type", "application/json")
	respWebhook := httptest.NewRecorder()
	router.ServeHTTP(respWebhook, reqWebhook)

	if respWebhook.Code != http.StatusOK {
		t.Fatalf("webhook status = %d, want 200", respWebhook.Code)
	}

	// 6. Check status API after webhook -> must be is_paid: true
	reqStatusPaid := httptest.NewRequest(http.MethodGet, "/api/pesanan/"+orderCode+"/status", nil)
	respStatusPaid := httptest.NewRecorder()
	router.ServeHTTP(respStatusPaid, reqStatusPaid)

	if !strings.Contains(respStatusPaid.Body.String(), `"is_paid":true`) {
		t.Errorf("order status after webhook should be is_paid: true, got: %s", respStatusPaid.Body.String())
	}
}
