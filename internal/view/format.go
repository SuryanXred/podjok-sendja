package view

import (
	"fmt"
	"strconv"
)

func FormatCategoryIndex(index int) string {
	return fmt.Sprintf("%02d", index+1)
}

func FormatRupiah(amount int64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}

	digits := strconv.FormatInt(amount, 10)
	formatted := make([]byte, 0, len(digits)+len(digits)/3)
	for i := 0; i < len(digits); i++ {
		if i > 0 && (len(digits)-i)%3 == 0 {
			formatted = append(formatted, '.')
		}
		formatted = append(formatted, digits[i])
	}

	return "Rp " + sign + string(formatted)
}
