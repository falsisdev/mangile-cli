package formats

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Mushoku Tensei", "mushoku-tensei"},
		{"Bilim Kurgu", "bilim-kurgu"},
		{"İstanbul!", "istanbul"},
		{"Şeytan Çıkarma", "seytan-cikarma"},
		{"Ölümüne", "olumune"},
		{"   ", "seri"},
	}
	for _, tt := range tests {
		if got := Slugify(tt.in); got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
