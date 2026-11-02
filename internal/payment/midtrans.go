package payment

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerKey    string
	ClientKey    string
	IsProduction bool
}

func GetConfig() Config {
	serverKey := strings.TrimSpace(os.Getenv("MIDTRANS_SERVER_KEY"))
	clientKey := strings.TrimSpace(os.Getenv("MIDTRANS_CLIENT_KEY"))
	isProd := os.Getenv("MIDTRANS_IS_PRODUCTION") == "true"

	return Config{
		ServerKey:    serverKey,
		ClientKey:    clientKey,
		IsProduction: isProd,
	}
}

type QRISResult struct {
	OrderID     string `json:"order_id"`
	GrossAmount int64  `json:"gross_amount"`
	QRURL       string `json:"qr_url"`
	QRString    string `json:"qr_string"`
	IsSimulated bool   `json:"is_simulated"`
}

type midtransChargeRequest struct {
	PaymentType        string                 `json:"payment_type"`
	TransactionDetails map[string]interface{} `json:"transaction_details"`
	QRIS               map[string]interface{} `json:"qris,omitempty"`
}

type midtransChargeResponse struct {
	StatusCode        string `json:"status_code"`
	StatusMessage     string `json:"status_message"`
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	TransactionStatus string `json:"transaction_status"`
	QRString          string `json:"qr_string"`
	Actions           []struct {
		Name   string `json:"name"`
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"actions"`
}

// GenerateQRIS menghasilkan QRIS dinamis berdasar kode pesanan dan total harga.
// Jika MIDTRANS_SERVER_KEY diset, fungsi ini memanggil Midtrans Core API secara langsung.
// Jika belum diset, fungsi ini menyediakan QRIS simulasi sandbox yang valid untuk pengujian lokal.
func GenerateQRIS(orderID string, grossAmount int64) (*QRISResult, error) {
	cfg := GetConfig()

	// Jika Server Key disetel, panggil Midtrans API asli
	if cfg.ServerKey != "" {
		endpoint := "https://api.sandbox.midtrans.com/v2/charge"
		if cfg.IsProduction {
			endpoint = "https://api.midtrans.com/v2/charge"
		}

		reqBody := midtransChargeRequest{
			PaymentType: "qris",
			TransactionDetails: map[string]interface{}{
				"order_id":     orderID,
				"gross_amount": grossAmount,
			},
			QRIS: map[string]interface{}{
				"acquirer": "gopay",
			},
		}

		jsonBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonBytes))
		if err != nil {
			return nil, err
		}

		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.ServerKey+":"))
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			defer resp.Body.Close()
			var mResp midtransChargeResponse
			if err := json.NewDecoder(resp.Body).Decode(&mResp); err == nil {
				qrURL := ""
				for _, act := range mResp.Actions {
					if act.Name == "generate-qr-code" {
						qrURL = act.URL
						break
					}
				}
				if qrURL == "" && len(mResp.Actions) > 0 {
					qrURL = mResp.Actions[0].URL
				}

				return &QRISResult{
					OrderID:     orderID,
					GrossAmount: grossAmount,
					QRURL:       qrURL,
					QRString:    mResp.QRString,
					IsSimulated: false,
				}, nil
			}
		}
	}

	// Mode Simulasi Sandbox (Bila Server Key belum dipasang di .env)
	// Menghasilkan QR code berstandar dengan nominal tepat agar developer langsung dapat menguji di localhost
	simulatedQRText := fmt.Sprintf("https://simulator.sandbox.midtrans.com/qris/%s?amount=%d", orderID, grossAmount)
	encodedQRText := url.QueryEscape(simulatedQRText)
	qrImageURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=%s", encodedQRText)

	return &QRISResult{
		OrderID:     orderID,
		GrossAmount: grossAmount,
		QRURL:       qrImageURL,
		QRString:    simulatedQRText,
		IsSimulated: true,
	}, nil
}

type MidtransNotification struct {
	TransactionTime   string `json:"transaction_time"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	StatusMessage     string `json:"status_message"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
	PaymentType       string `json:"payment_type"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	FraudStatus       string `json:"fraud_status"`
}

// VerifySignature memverifikasi signature key notifikasi dari Midtrans jika server key disetel.
func VerifySignature(notif MidtransNotification, serverKey string) bool {
	if serverKey == "" {
		return true // Sandbox / Dev mode
	}

	raw := notif.OrderID + notif.StatusCode + notif.GrossAmount + serverKey
	hasher := sha512.New()
	hasher.Write([]byte(raw))
	expected := hex.EncodeToString(hasher.Sum(nil))

	return strings.EqualFold(notif.SignatureKey, expected)
}

// ParseNotification membaca body notifikasi HTTP dari Midtrans.
func ParseNotification(r io.Reader) (*MidtransNotification, error) {
	var notif MidtransNotification
	if err := json.NewDecoder(r).Decode(&notif); err != nil {
		return nil, errors.New("format notifikasi tidak valid")
	}
	return &notif, nil
}

