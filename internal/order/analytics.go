package order

import (
	"sort"
	"strings"
)

type ProductSalesMetric struct {
	ProductName string
	Quantity    int
	TotalSales  int64
}

type AdminReport struct {
	Metrics           StoreMetrics
	AverageOrderValue int64
	CashOrders        int
	QRISOrders        int
	MahasiswaOrders   int
	UmumOrders        int
	TopProducts       []ProductSalesMetric
	RecentOrders      []*Order
}

type CashierMetrics struct {
	TotalOrders    int
	PendingPayment int
	InProcess      int
	ReadyForPickup int
	Completed      int
	Cancelled      int
}

func (s *Store) GetCashierMetrics() CashierMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var m CashierMetrics
	m.TotalOrders = len(s.orders)
	for _, o := range s.orders {
		switch o.OrderStatus {
		case OrderStatusPendingPayment:
			m.PendingPayment++
		case OrderStatusPaid, OrderStatusPreparing:
			m.InProcess++
		case OrderStatusReady:
			m.ReadyForPickup++
		case OrderStatusCompleted:
			m.Completed++
		case OrderStatusCancelled:
			m.Cancelled++
		}
	}
	return m
}

func (s *Store) GetAdminReport() AdminReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var r AdminReport
	r.Metrics = s.GetMetrics()

	productMap := make(map[string]*ProductSalesMetric)
	var completedOrPaidCount int

	for _, o := range s.orders {
		if o.PaymentMethod == PaymentMethodCash {
			r.CashOrders++
		} else if o.PaymentMethod == PaymentMethodQRIS {
			r.QRISOrders++
		}

		if o.CustomerType == "mahasiswa" {
			r.MahasiswaOrders++
		} else {
			r.UmumOrders++
		}

		if o.PaymentStatus == PaymentStatusPaid || o.OrderStatus == OrderStatusCompleted {
			completedOrPaidCount++
			for _, it := range o.Items {
				itemKey := it.ProductName
				if it.Variant != "" {
					itemKey = it.ProductName + " (" + it.Variant + ")"
				}
				if _, exists := productMap[itemKey]; !exists {
					productMap[itemKey] = &ProductSalesMetric{
						ProductName: itemKey,
					}
				}
				productMap[itemKey].Quantity += it.Quantity
				productMap[itemKey].TotalSales += it.Subtotal
			}
		}
	}

	if completedOrPaidCount > 0 && r.Metrics.ConfirmedSales > 0 {
		r.AverageOrderValue = r.Metrics.ConfirmedSales / int64(completedOrPaidCount)
	}

	topList := make([]ProductSalesMetric, 0, len(productMap))
	for _, p := range productMap {
		topList = append(topList, *p)
	}

	sort.Slice(topList, func(i, j int) bool {
		if topList[i].Quantity == topList[j].Quantity {
			return topList[i].TotalSales > topList[j].TotalSales
		}
		return topList[i].Quantity > topList[j].Quantity
	})

	if len(topList) > 8 {
		topList = topList[:8]
	}
	r.TopProducts = topList

	// 10 pesanan terbaru
	allOrders := make([]*Order, 0, len(s.orders))
	for _, o := range s.orders {
		allOrders = append(allOrders, o)
	}
	sort.Slice(allOrders, func(i, j int) bool {
		return allOrders[i].CreatedAt.After(allOrders[j].CreatedAt)
	})

	if len(allOrders) > 10 {
		r.RecentOrders = allOrders[:10]
	} else {
		r.RecentOrders = allOrders
	}

	return r
}

func (s *Store) SearchOrders(query string, filterStatus OrderStatus) []*Order {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	result := make([]*Order, 0)

	for _, o := range s.orders {
		if filterStatus != "" && o.OrderStatus != filterStatus {
			continue
		}

		if q != "" {
			codeMatch := strings.Contains(strings.ToLower(o.Code), q)
			nameMatch := strings.Contains(strings.ToLower(o.CustomerName), q)
			phoneMatch := strings.Contains(strings.ToLower(o.CustomerPhone), q)
			nimMatch := strings.Contains(strings.ToLower(o.CustomerNIM), q)
			if !codeMatch && !nameMatch && !phoneMatch && !nimMatch {
				continue
			}
		}

		result = append(result, o)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}
