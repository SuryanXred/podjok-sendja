package catalog

import "testing"

func TestSeedHasCategoriesInDisplayOrder(t *testing.T) {
	got := Seed()
	want := []string{"Coffee", "Tea & Milk", "Food", "Snack"}

	if len(got) != len(want) {
		t.Fatalf("category count = %d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("category %d name = %q, want %q", i, got[i].Name, name)
		}
	}
}

func TestSeedHasSeventeenProducts(t *testing.T) {
	count := 0
	for _, category := range Seed() {
		count += len(category.Products)
	}
	if count != 17 {
		t.Fatalf("product count = %d, want 17", count)
	}
}

func TestVariantPrices(t *testing.T) {
	tests := []struct {
		name    string
		slug    string
		variant string
		want    int64
	}{
		{"Americano ice", "americano", "Ice", 15000},
		{"Americano hot", "americano", "Hot", 12000},
		{"Espresso", "espresso", "", 10000},
		{"Caramel ice", "caramel", "Ice", 18000},
		{"Caramel hot", "caramel", "Hot", 15000},
		{"Vanilla Latte ice", "vanilla-latte", "Ice", 18000},
		{"Vanilla Latte hot", "vanilla-latte", "Hot", 15000},
		{"Brown Sugar ice", "brown-sugar", "Ice", 18000},
		{"Brown Sugar hot", "brown-sugar", "Hot", 15000},
		{"Spanish Latte ice", "spanish-latte", "Ice", 20000},
		{"Spanish Latte hot", "spanish-latte", "Hot", 18000},
		{"Ginger Tea ice", "ginger-tea", "Ice", 12000},
		{"Ginger Tea hot", "ginger-tea", "Hot", 7000},
		{"Lemongrass ice", "lemongrass", "Ice", 12000},
		{"Lemongrass hot", "lemongrass", "Hot", 7000},
		{"Choco Milk", "choco-milk", "", 18000},
		{"Strawberry Milk", "strawberry-milk", "", 18000},
		{"Matcha Latte", "matcha-latte", "", 18000},
		{"Mie Aneka Rasa", "mie-aneka-rasa", "", 7000},
		{"Mie Rebus Spesial", "mie-rebus-spesial", "", 12000},
		{"Mie Goreng Spesial", "mie-goreng-spesial", "", 12000},
		{"Telur", "telur", "", 4000},
		{"French Fries", "french-fries", "", 12000},
		{"Mix Platter", "mix-platter", "", 20000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product, ok := findProduct(tt.slug)
			if !ok {
				t.Fatalf("product %q not found", tt.slug)
			}
			got, err := product.PriceFor(tt.variant)
			if err != nil {
				t.Fatalf("PriceFor(%q) error: %v", tt.variant, err)
			}
			if got != tt.want {
				t.Errorf("PriceFor(%q) = %d, want %d", tt.variant, got, tt.want)
			}
		})
	}
}

func TestPriceForRejectsUnknownVariant(t *testing.T) {
	product, ok := findProduct("americano")
	if !ok {
		t.Fatal("americano not found")
	}

	if _, err := product.PriceFor("Extra Hot"); err == nil {
		t.Fatal("expected error for unknown variant")
	}
}

func TestPriceForRequiresVariantWhenProductHasVariants(t *testing.T) {
	product, ok := findProduct("americano")
	if !ok {
		t.Fatal("americano not found")
	}

	if _, err := product.PriceFor(""); err == nil {
		t.Fatal("expected error when variant is empty")
	}
}

func TestPriceForRejectsVariantOnSingleProduct(t *testing.T) {
	product, ok := findProduct("espresso")
	if !ok {
		t.Fatal("espresso not found")
	}

	if _, err := product.PriceFor("Ice"); err == nil {
		t.Fatal("expected error for variant on single product")
	}
}

func TestSlugsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, category := range Seed() {
		if seen[category.Slug] {
			t.Errorf("duplicate category slug %q", category.Slug)
		}
		seen[category.Slug] = true
		for _, product := range category.Products {
			if seen[product.Slug] {
				t.Errorf("duplicate product slug %q", product.Slug)
			}
			seen[product.Slug] = true
		}
	}
}

func findProduct(slug string) (Product, bool) {
	for _, category := range Seed() {
		for _, product := range category.Products {
			if product.Slug == slug {
				return product, true
			}
		}
	}
	return Product{}, false
}
