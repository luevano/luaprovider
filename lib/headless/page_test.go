package headless

import "testing"

func TestCheckURL(t *testing.T) {
	for url, ok := range map[string]bool{
		"https://example.com/a":            true,
		"HTTP://example.com":               true,
		"file:///home/user/mangal.toml":    false,
		"FILE:///etc/passwd":               false,
		"/etc/passwd":                      false,
		"chrome://settings":                false,
		"javascript:alert(1)":              false,
		"data:text/html,<script></script>": false,
		" file:///etc/passwd":              false,
	} {
		if err := checkURL(url); (err == nil) != ok {
			t.Errorf("%q: got %v, want ok=%v", url, err, ok)
		}
	}
}
