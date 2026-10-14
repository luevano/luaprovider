package html

import (
	"net/url"
	"testing"
)

const page = `<html><body><nav><a href="/">Home</a></nav><article>
<p>Gol D. Roger was known as the Pirate King, the strongest and most infamous being to have sailed the Grand Line. The capture and death of Roger by the World Government brought a change throughout the world.</p>
<p>See <a href="/chapters">chapters</a> and <img src="cover.jpg"></p>
</article></body></html>`

func TestSimplifyResolvesLinksAgainstTheKnownURL(t *testing.T) {
	for _, c := range []struct {
		pageURL   string
		href, src string
	}{
		{"", "/chapters", "cover.jpg"},
		{"https://site.com/manga/x", "https://site.com/chapters", "https://site.com/manga/cover.jpg"},
	} {
		var pageURL *url.URL
		if c.pageURL != "" {
			pageURL, _ = url.Parse(c.pageURL)
		}
		doc, err := simplify(page, pageURL)
		if err != nil {
			t.Fatal(err)
		}
		if href := doc.Find("a").AttrOr("href", ""); href != c.href {
			t.Errorf("url %q: href = %q, want %q", c.pageURL, href, c.href)
		}
		if src := doc.Find("img").AttrOr("src", ""); src != c.src {
			t.Errorf("url %q: src = %q, want %q", c.pageURL, src, c.src)
		}
	}
}

func TestSimplifyEmpty(t *testing.T) {
	doc, err := simplify("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if text := doc.Text(); text != "" {
		t.Errorf("text = %q", text)
	}
}
