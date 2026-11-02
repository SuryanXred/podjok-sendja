package view

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"caffe-podjok-sendja/internal/customer"
	"caffe-podjok-sendja/internal/order"
)

func TestAccountPageRendersEmailWithFadeClasses(t *testing.T) {
	cust := &customer.Customer{
		ID:        "cust-123",
		Name:      "Budi Santoso",
		Phone:     "081234567890",
		Email:     "budisantoso_mahasiswapanjang12345@mahasiswa.unism.ac.id",
		Type:      customer.CustomerTypeUmum,
		CreatedAt: time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC),
	}

	var output bytes.Buffer
	if err := AccountPage(cust, []*order.Order{}).Render(context.Background(), &output); err != nil {
		t.Fatalf("render account page: %v", err)
	}

	html := output.String()
	if !strings.Contains(html, "profile-detail-email") {
		t.Error("account page does not contain profile-detail-email container")
	}
	if !strings.Contains(html, "profile-email-value") {
		t.Error("account page does not contain profile-email-value element")
	}
	if !strings.Contains(html, `title="budisantoso_mahasiswapanjang12345@mahasiswa.unism.ac.id"`) {
		t.Error("account page does not have title attribute with full email")
	}
	if !strings.Contains(html, "Terdaftar Sejak") {
		t.Error("account page missing Terdaftar Sejak label")
	}
}
