package httpapi

import "caffe-podjok-sendja/internal/view"

func formatRupiah(amount int64) string {
	return view.FormatRupiah(amount)
}
