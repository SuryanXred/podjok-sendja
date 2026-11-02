package view

import "testing"

func TestFormatCategoryIndex(t *testing.T) {
	tests := []struct {
		index int
		want  string
	}{
		{0, "01"},
		{1, "02"},
		{9, "10"},
	}

	for _, tt := range tests {
		if got := FormatCategoryIndex(tt.index); got != tt.want {
			t.Errorf("FormatCategoryIndex(%d) = %q, want %q", tt.index, got, tt.want)
		}
	}
}
