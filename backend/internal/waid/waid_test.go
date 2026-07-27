package waid

import "testing"

func TestNormalizeValid(t *testing.T) {
	cases := []struct{ in, want string }{
		{"081231372105", "6281231372105"},
		{"+62 812-3137-2105", "6281231372105"},
		{"6281231372105@c.us", "6281231372105"},
		{"6281231372105@s.whatsapp.net", "6281231372105"},
		{"81231372105", "6281231372105"},
		{"  081231372105  ", "6281231372105"},
		{"(0812) 3137 2105", "6281231372105"},
		{"6281231372105:12@c.us", "6281231372105"},
	}
	for _, c := range cases {
		got, err := Normalize(c.in)
		if err != nil {
			t.Errorf("Normalize(%q) error tak terduga: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("Normalize(%q) = %q, mau %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeInvalid(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"abc",
		"62812",                     // terlalu pendek
		"120363123456789012@g.us",   // JID grup, bukan nomor
		"12345678901234567890",      // terlalu panjang
		"+62 0812-3137-2105",        // salah ketik: kode negara + trunk "0" dobel
		"021-5551234",               // nomor telepon rumah (landline), bukan seluler
	}
	for _, in := range cases {
		got, err := Normalize(in)
		if err == nil {
			t.Errorf("Normalize(%q) = %q, mau error", in, got)
		}
	}
}
