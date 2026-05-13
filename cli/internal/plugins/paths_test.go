package plugins

import "testing"

func TestSanitizeSegment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"acme", "acme"},
		{"Acme.Corp_1", "Acme.Corp_1"},
		{"path/with/slash", "path_with_slash"},
		// Если строка состояла только из недопустимых символов и схлопнулась в подчёркивания,
		// возвращаем единственный _ - это лучше чем многоподчёркнутые папки в файловой системе.
		{"привет", "_"},
		{" trim me ", "trim_me"},
		{"", "_"},
		{"   ", "_"},
		{"//", "_"},
		{"a..b", "a..b"},
		{"v0.1.0-rc1", "v0.1.0-rc1"},
	}

	for _, tc := range cases {
		got := sanitizeSegment(tc.in)
		if got != tc.want {
			t.Errorf("sanitizeSegment(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStoreDirShape(t *testing.T) {
	t.Parallel()
	cfg := Config{Root: "/tmp/test-root"}
	got := storeDir(cfg, "Acme", "observability.prom", "0.1.0", "darwin", "arm64")
	want := "/tmp/test-root/plugins/store/Acme/observability.prom/0.1.0/darwin-arm64"
	if got != want {
		t.Errorf("storeDir = %q, want %q", got, want)
	}
}
