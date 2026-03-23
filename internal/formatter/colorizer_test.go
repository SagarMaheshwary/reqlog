package formatter

import (
	"hash/fnv"
	"testing"
)

func TestNewColorizer(t *testing.T) {
	c := NewColorizer()
	if c == nil {
		t.Fatal("expected non-nil colorizer")
	}
}

func TestColorizer_Color_SameServiceGetsSameColor(t *testing.T) {
	c := NewColorizer()

	first := c.Color("auth-service")
	second := c.Color("auth-service")

	if first != second {
		t.Fatalf("expected same service to get same color, got %q and %q", first, second)
	}
}

func TestColorizer_Color_ReturnsColorFromPalette(t *testing.T) {
	c := NewColorizer()

	got := c.Color("payment-service")

	if !contains(colors, got) {
		t.Fatalf("expected color %q to exist in palette", got)
	}
}

func TestColorizer_Color_IsDeterministicAcrossInstances(t *testing.T) {
	c1 := NewColorizer()
	c2 := NewColorizer()

	got1 := c1.Color("order-service")
	got2 := c2.Color("order-service")

	if got1 != got2 {
		t.Fatalf("expected deterministic color across instances, got %q and %q", got1, got2)
	}
}

func TestFnv32a(t *testing.T) {
	text := "auth-service"

	h := fnv.New32a()
	_, err := h.Write([]byte(text))
	if err != nil {
		t.Fatalf("unexpected error writing hash input: %v", err)
	}
	want := h.Sum32()

	got := fnv32a(text)

	if got != want {
		t.Fatalf("expected fnv32a(%q) = %d, got %d", text, want, got)
	}
}

func TestFnv32a_EmptyString(t *testing.T) {
	h := fnv.New32a()
	want := h.Sum32()

	got := fnv32a("")

	if got != want {
		t.Fatalf("expected fnv32a(empty) = %d, got %d", want, got)
	}
}
