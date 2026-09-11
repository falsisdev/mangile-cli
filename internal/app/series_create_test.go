package app

import (
	"testing"
)

func TestMergeTags(t *testing.T) {
	got := mergeTags([]string{"Aksiyon"}, []string{"Aksiyon", "Dram", "Uydurma"})
	if len(got) != 2 || got[0] != "Aksiyon" || got[1] != "Dram" {
		t.Errorf("birleştirme yanlış: %v", got)
	}
	if got := mergeTags(nil, nil); len(got) != 0 {
		t.Errorf("boş giriş boş çıkmalı: %v", got)
	}
}

func TestEqualTags(t *testing.T) {
	if !equalTags([]string{"Aksiyon"}, []string{"Aksiyon"}) {
		t.Error("aynı liste eşit olmalı")
	}
	if equalTags([]string{"Aksiyon"}, []string{"Dram"}) {
		t.Error("farklı liste eşit olmamalı")
	}
	if equalTags([]string{"Aksiyon"}, []string{"Aksiyon", "Dram"}) {
		t.Error("uzunluk farkı eşit olmamalı")
	}
}
