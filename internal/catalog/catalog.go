package catalog

import "errors"

var (
	ErrVariantRequired = errors.New("variant is required")
	ErrUnknownVariant  = errors.New("unknown variant")
)

type Category struct {
	Slug      string
	Name      string
	SortOrder int
	Products  []Product
}

type Product struct {
	Slug      string
	Name      string
	BasePrice int64
	Variants  []Variant
}

func (c Category) ImagePath() string {
	switch c.Slug {
	case "coffee":
		return "/static/images/categories/Coffee.jpeg"
	case "tea-and-milk":
		return "/static/images/categories/tea-and-milk.jpeg"
	case "food":
		return "/static/images/categories/Food.jpeg"
	case "snack":
		return "/static/images/categories/Snack.jpeg"
	default:
		return ""
	}
}

func (p Product) ImagePath() string {
	return "/static/images/products/" + p.Slug + ".webp"
}

type Variant struct {
	Name       string
	PriceDelta int64
}

func (p Product) HasVariants() bool {
	return len(p.Variants) > 0
}

func (p Product) PriceFor(variant string) (int64, error) {
	if !p.HasVariants() {
		if variant != "" {
			return 0, ErrUnknownVariant
		}
		return p.BasePrice, nil
	}

	if variant == "" {
		return 0, ErrVariantRequired
	}

	for _, candidate := range p.Variants {
		if candidate.Name == variant {
			return p.BasePrice + candidate.PriceDelta, nil
		}
	}

	return 0, ErrUnknownVariant
}

func Seed() []Category {
	return []Category{
		{
			Slug:      "coffee",
			Name:      "Coffee",
			SortOrder: 1,
			Products: []Product{
				{Slug: "americano", Name: "Americano", BasePrice: 12000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 3000},
					{Name: "Hot", PriceDelta: 0},
				}},
				{Slug: "espresso", Name: "Espresso", BasePrice: 10000},
				{Slug: "caramel", Name: "Caramel", BasePrice: 15000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 3000},
					{Name: "Hot", PriceDelta: 0},
				}},
				{Slug: "vanilla-latte", Name: "Vanilla Latte", BasePrice: 15000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 3000},
					{Name: "Hot", PriceDelta: 0},
				}},
				{Slug: "brown-sugar", Name: "Brown Sugar", BasePrice: 15000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 3000},
					{Name: "Hot", PriceDelta: 0},
				}},
				{Slug: "spanish-latte", Name: "Spanish Latte", BasePrice: 18000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 2000},
					{Name: "Hot", PriceDelta: 0},
				}},
			},
		},
		{
			Slug:      "tea-and-milk",
			Name:      "Tea & Milk",
			SortOrder: 2,
			Products: []Product{
				{Slug: "ginger-tea", Name: "Ginger Tea", BasePrice: 7000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 5000},
					{Name: "Hot", PriceDelta: 0},
				}},
				{Slug: "lemongrass", Name: "Lemongrass", BasePrice: 7000, Variants: []Variant{
					{Name: "Ice", PriceDelta: 5000},
					{Name: "Hot", PriceDelta: 0},
				}},
				{Slug: "choco-milk", Name: "Choco Milk", BasePrice: 18000},
				{Slug: "strawberry-milk", Name: "Strawberry Milk", BasePrice: 18000},
				{Slug: "matcha-latte", Name: "Matcha Latte", BasePrice: 18000},
			},
		},
		{
			Slug:      "food",
			Name:      "Food",
			SortOrder: 3,
			Products: []Product{
				{Slug: "mie-aneka-rasa", Name: "Mie Rebus/Goreng Aneka Rasa", BasePrice: 7000},
				{Slug: "mie-rebus-spesial", Name: "Mie Rebus Spesial", BasePrice: 12000},
				{Slug: "mie-goreng-spesial", Name: "Mie Goreng Spesial", BasePrice: 12000},
				{Slug: "telur", Name: "Telur Rebus/Goreng", BasePrice: 4000},
			},
		},
		{
			Slug:      "snack",
			Name:      "Snack",
			SortOrder: 4,
			Products: []Product{
				{Slug: "french-fries", Name: "French Fries", BasePrice: 12000},
				{Slug: "mix-platter", Name: "Mix Platter (Sosis dan Kentang)", BasePrice: 20000},
			},
		},
	}
}
