package payment

import (
	"strings"
	"testing"
)

func TestGenerateQRISSimulated(t *testing.T) {
	result, err := GenerateQRIS("PDS-261028-0001", 27000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OrderID != "PDS-261028-0001" {
		t.Errorf("OrderID = %q, want PDS-261028-0001", result.OrderID)
	}
	if result.GrossAmount != 27000 {
		t.Errorf("GrossAmount = %d, want 27000", result.GrossAmount)
	}
	if !strings.Contains(result.QRURL, "create-qr-code") {
		t.Errorf("QRURL %q should point to QR image", result.QRURL)
	}
	if !strings.Contains(result.QRString, "simulator.sandbox.midtrans.com") {
		t.Errorf("QRString %q should contain simulator URL", result.QRString)
	}
}

func TestVerifySignature(t *testing.T) {
	notif := MidtransNotification{
		OrderID:           "PDS-001",
		StatusCode:        "200",
		GrossAmount:       "27000.00",
		SignatureKey:      "",
		TransactionStatus: "settlement",
	}

	// Empty server key = dev mode, always valid
	if !VerifySignature(notif, "") {
		t.Error("empty server key should return true for dev mode")
	}
}

func TestParseNotification(t *testing.T) {
	rawJSON := `{
		"order_id": "PDS-261028-0001",
		"status_code": "200",
		"gross_amount": "27000.00",
		"transaction_status": "settlement",
		"payment_type": "qris"
	}`

	notif, err := ParseNotification(strings.NewReader(rawJSON))
	if err != nil {
		t.Fatalf("failed to parse notification: %v", err)
	}

	if notif.OrderID != "PDS-261028-0001" {
		t.Errorf("OrderID = %q, want PDS-261028-0001", notif.OrderID)
	}
	if notif.TransactionStatus != "settlement" {
		t.Errorf("TransactionStatus = %q, want settlement", notif.TransactionStatus)
	}
}

