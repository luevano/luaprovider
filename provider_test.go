package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/progress"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/syncmap"
)

const testScript = `
function SearchMangas(query)
	if query == "nil" then return nil end
	return {{
		id = "1", title = "Manga " .. query, url = "https://site/1",
		metadata = {summary = "about it", author_list = {"someone"}, date_start = "2020-01-02", publication_status = "RELEASING"},
	}}
end

function MangaVolumes(manga)
	return {{number = 1}}
end

function VolumeChapters(volume)
	return {
		{title = "Prologue"},
		{title = "One", number = 1, date = "2021-02-03"},
		{title = "Extra"},
		{title = "Official", number = 2, unavailable = "external"},
	}
end

function ChapterPages(chapter)
	return {{url = "https://site/1.png", extension = ".png"}}
end
`

// logged sets a libmangal logger on p and returns what it logged, starting with the backlog.
func logged(p *provider) []string {
	var msgs []string
	l := logger.NewLogger()
	l.SetOnLog(func(format string, a ...any) { msgs = append(msgs, fmt.Sprintf(format, a...)) })
	p.SetLogger(l)
	return msgs
}

func loadTestProvider(t *testing.T, script string) *provider {
	t.Helper()
	return loadTestProviderWith(t, script, DefaultOptions())
}

func loadTestProviderWith(t *testing.T, script string, options Options) *provider {
	t.Helper()
	info := libmangal.ProviderInfo{ID: "test", Name: "Test", Version: "0.1.0"}
	loader, err := NewLoader([]byte(script), info, options)
	if err != nil {
		t.Fatal(err)
	}
	p, err := loader.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	if p, ok := p.(mangaByIDProvider); ok {
		return p.provider
	}
	return p.(*provider)
}

func TestNonTableReturnIsAnError(t *testing.T) {
	p := loadTestProvider(t, testScript)

	_, err := p.SearchMangas(context.Background(), "nil")
	if err == nil || !strings.Contains(err.Error(), `test: search "nil": expected a table, got nil`) {
		t.Errorf("wanted an error, got %v", err)
	}
}

func TestManyCallsDontGrowTheStack(t *testing.T) {
	p := loadTestProvider(t, testScript)
	before := p.state.GetTop()

	for i := range 10000 {
		if _, err := p.SearchMangas(context.Background(), "x"); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if top := p.state.GetTop(); top != before {
		t.Errorf("stack grew from %d to %d values", before, top)
	}
}

func TestConcurrentCalls(t *testing.T) {
	p := loadTestProvider(t, testScript)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if _, err := p.SearchMangas(context.Background(), "x"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}

func TestChaptersAndMetadata(t *testing.T) {
	options := DefaultOptions()
	options.Filter.ShowUnavailableChapters = true
	p := loadTestProviderWith(t, testScript, options)
	ctx := context.Background()

	mangas, err := p.SearchMangas(ctx, "x")
	if err != nil {
		t.Fatal(err)
	}
	meta := mangas[0].Metadata().(*mangadata.Metadata)
	if meta.Summary != "about it" || meta.Title() != "Manga x" || meta.DateStart.Year != 2020 || meta.PublicationStatus != "RELEASING" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	volumes, err := p.MangaVolumes(ctx, mangas[0])
	if err != nil {
		t.Fatal(err)
	}
	chapters, err := p.VolumeChapters(ctx, volumes[0])
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		number      float32
		unavailable string
	}{{0.5, ""}, {1, ""}, {1.5, ""}, {2, "external"}}
	if len(chapters) != len(want) {
		t.Fatalf("wanted %d chapters, got %d", len(want), len(chapters))
	}
	for i, chapter := range chapters {
		info := chapter.Info()
		if info.Number != want[i].number || info.Unavailable != want[i].unavailable {
			t.Errorf("chapter %q: wanted %v %q, got %v %q", info.Title, want[i].number, want[i].unavailable, info.Number, info.Unavailable)
		}
	}
	if date := chapters[1].Info().Date; date.Year != 2021 || date.Month != 2 || date.Day != 3 {
		t.Errorf("wrong date: %+v", date)
	}

	pages, err := p.ChapterPages(ctx, chapters[1])
	if err != nil {
		t.Fatal(err)
	}
	if page := pages[0].(*luaPage); page.Headers["User-Agent"] == "" {
		t.Error("pages should get the default user agent")
	}
	var _ mangadata.Page = pages[0]
}

func TestLoadErrors(t *testing.T) {
	info := libmangal.ProviderInfo{ID: "test", Name: "Test", Version: "0.1.0"}
	for script, want := range map[string]string{
		`error("broken")`:             "load test:",
		`function SearchMangas() end`: "load test: missing function",
	} {
		loader, err := NewLoader([]byte(script), info, DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loader.Load(context.Background()); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: wanted %q, got %v", script, want, err)
		}
	}
}

func TestJSRunStopsOnCancel(t *testing.T) {
	p := loadTestProvider(t, `
local sdk = require("sdk")
function SearchMangas(query)
	sdk.js.vm():run("while (true) {}")
	return {}
end
function MangaVolumes() return {} end
function VolumeChapters() return {} end
function ChapterPages() return {} end
`)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := p.SearchMangas(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("wanted the deadline error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("should stop right after the deadline, took %s", elapsed)
	}
}

func TestScriptLog(t *testing.T) {
	p := loadTestProvider(t, `
local sdk = require("sdk")
sdk.log.print("loaded")
function SearchMangas() return {} end
function MangaVolumes() return {} end
function VolumeChapters() return {} end
function ChapterPages() return {} end
`)
	if got := logged(p); len(got) != 1 || got[0] != "[backlog] test: loaded" {
		t.Errorf("wanted the script's message in the backlog, got %q", got)
	}
}

func TestUnavailableChaptersAreHidden(t *testing.T) {
	options := DefaultOptions()
	options.Filter.ShowUnavailableChapters = false
	p := loadTestProviderWith(t, testScript, options)
	ctx := context.Background()

	mangas, err := p.SearchMangas(ctx, "x")
	if err != nil {
		t.Fatal(err)
	}
	volumes, err := p.MangaVolumes(ctx, mangas[0])
	if err != nil {
		t.Fatal(err)
	}
	chapters, err := p.VolumeChapters(ctx, volumes[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, chapter := range chapters {
		if chapter.Info().Unavailable != "" {
			t.Errorf("chapter %q should be hidden", chapter)
		}
	}
	if len(chapters) != 3 {
		t.Errorf("wanted 3 chapters, got %d", len(chapters))
	}
}

func TestScriptOptions(t *testing.T) {
	options := DefaultOptions()
	options.Filter.Language = "es"
	options.Filter.NSFW = true
	p := loadTestProviderWith(t, `
local options = require("sdk").options
assert(options.language == "es" and options.nsfw, "options")
`+testScript, options)
	if p == nil {
		t.Fatal("not loaded")
	}
}

// countingStore counts how many times it's open.
type countingStore struct {
	gokv.Store
	open *int
}

func (s countingStore) Close() error {
	*s.open--
	return nil
}

func TestCacheStoreOpenedOnce(t *testing.T) {
	var open, opens int
	store := syncmap.NewStore(syncmap.DefaultOptions)
	options := DefaultOptions()
	options.CacheStore = func(string, string) (gokv.Store, error) {
		open++
		opens++
		return countingStore{Store: store, open: &open}, nil
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hi")
	}))
	defer server.Close()

	p := loadTestProviderWith(t, fmt.Sprintf(`
local sdk = require("sdk")
sdk.http.request("GET", %q):send()
sdk.http.request("GET", %q):send()
`, server.URL, server.URL)+testScript, options)
	if opens != 1 || open != 1 {
		t.Errorf("opened %d times (%d still open) after loading, want 1 (1)", opens, open)
	}
	p.Close()
	if open != 0 {
		t.Errorf("still open %d times after closing the provider", open)
	}
}

func TestMangaByID(t *testing.T) {
	info := libmangal.ProviderInfo{ID: "test", Name: "Test", Version: "0.1.0"}
	load := func(script string) libmangal.Provider {
		t.Helper()
		loader, err := NewLoader([]byte(script), info, DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		p, err := loader.Load(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { p.Close() })
		return p
	}

	if _, ok := load(testScript).(libmangal.MangaByIDProvider); ok {
		t.Error("a script without MangaByID shouldn't implement MangaByIDProvider")
	}

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	p, ok := load(testScript + fmt.Sprintf(`
local sdk = require("sdk")
function MangaByID(id)
	if id == "404" then
		sdk.http.request("GET", %q):send():check()
	elseif id == "fail" then
		error("broken")
	elseif id ~= "1" then
		return nil
	end
	return {id = id, title = "Manga " .. id}
end
`, server.URL)).(libmangal.MangaByIDProvider)
	if !ok {
		t.Fatal("a script with MangaByID should implement MangaByIDProvider")
	}
	ctx := context.Background()

	manga, found, err := p.MangaByID(ctx, "1")
	if err != nil || !found || manga.Info().Title != "Manga 1" {
		t.Fatalf("got %v, %v, %v", manga, found, err)
	}
	if volumes, err := p.(libmangal.Provider).MangaVolumes(ctx, manga); err != nil || len(volumes) != 1 {
		t.Errorf("volumes of the manga: %v, %v", volumes, err)
	}
	for _, id := range []string{"2", "404"} {
		if manga, found, err := p.MangaByID(ctx, id); err != nil || found || manga != nil {
			t.Errorf("id %s: %v, %v, %v, want not found", id, manga, found, err)
		}
	}
	if _, _, err := p.MangaByID(ctx, "fail"); err == nil || !strings.Contains(err.Error(), `test: manga "fail": `) {
		t.Errorf("error: %v", err)
	}
}

func TestLogStatus(t *testing.T) {
	p := loadTestProvider(t, `
local log = require("sdk").log
function SearchMangas(query)
	log.status("page 2 of 5")
	return {}
end
function MangaVolumes(manga) return {} end
function VolumeChapters(volume) return {} end
function ChapterPages(chapter) return {} end
`)
	var got []progress.Status
	ctx := progress.WithReporter(context.Background(), func(s progress.Status) { got = append(got, s) })
	if _, err := p.SearchMangas(ctx, "x"); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Source != "test" || got[0].Message != "Test: page 2 of 5" {
		t.Errorf("reported %+v", got)
	}
}

// failingStore fails to close.
type failingStore struct{ gokv.Store }

func (failingStore) Close() error { return errors.New("store close failed") }

func TestFailedLoadReportsCloseErrors(t *testing.T) {
	options := DefaultOptions()
	options.CacheStore = func(string, string) (gokv.Store, error) {
		return failingStore{syncmap.NewStore(syncmap.DefaultOptions)}, nil
	}
	info := libmangal.ProviderInfo{ID: "test", Name: "Test", Version: "0.1.0"}
	loader, err := NewLoader([]byte(`error("broken")`), info, options)
	if err != nil {
		t.Fatal(err)
	}
	_, err = loader.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "broken") || !strings.Contains(err.Error(), "store close failed") {
		t.Errorf("wanted the script and the close errors, got %v", err)
	}
}
