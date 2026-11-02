package view

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"caffe-podjok-sendja/internal/order"
)

func TestMenuPageShowsEmptyState(t *testing.T) {
	var output bytes.Buffer
	if err := MenuPage(nil, order.Cart{}, nil).Render(context.Background(), &output); err != nil {
		t.Fatalf("render menu page: %v", err)
	}

	if !strings.Contains(output.String(), "Menu belum tersedia") {
		t.Error("empty menu page does not show the empty state")
	}
}

func TestMenuPageEmptyStateStaysInPaperSheet(t *testing.T) {
	var output bytes.Buffer
	if err := MenuPage(nil, order.Cart{}, nil).Render(context.Background(), &output); err != nil {
		t.Fatalf("render menu page: %v", err)
	}

	html := output.String()
	sheetStart := strings.Index(html, `class="menu-sheet"`)
	if sheetStart == -1 {
		t.Fatal("empty menu has no paper sheet")
	}

	messageStart := strings.Index(html, `class="menu-empty"`)
	if messageStart == -1 {
		t.Fatal("empty menu message is missing")
	}
	if messageStart < sheetStart {
		t.Fatal("empty menu message is outside the paper sheet")
	}

	sheetClose := strings.Index(html[messageStart:], "</div>")
	if sheetClose == -1 {
		t.Fatal("paper sheet is not closed after the empty menu message")
	}
	if !strings.Contains(html[messageStart:messageStart+sheetClose], "Menu belum tersedia") {
		t.Fatal("empty menu message is outside the paper sheet")
	}
}
