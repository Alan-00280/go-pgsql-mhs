package service

import "strings"

// NormalizeTags merapikan tag sebelum disimpan:
// huruf kecil, tanpa spasi di tepi, tanpa duplikat,
// dan urutan pertama dipertahankan.
//
// Tanpa normalisasi, "Tugas-Banyak", "tugas-banyak", dan
// " tugas-banyak " tercatat sebagai tiga tag berbeda.
// Akibatnya, ringkasan tag terpopuler menjadi tidak bermakna.
func NormalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))

	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))

		if t == "" {
			continue
		}

		if _, dup := seen[t]; dup {
			continue
		}

		seen[t] = struct{}{}
		result = append(result, t)
	}

	return result
}
