package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/luevano/libmangal/logger"
	lua "github.com/yuin/gopher-lua"
)

func luaTable(t *testing.T, L *lua.LState, expr string) *lua.LTable {
	t.Helper()
	if err := L.DoString("return " + expr); err != nil {
		t.Fatal(err)
	}
	defer L.Pop(1)
	return L.CheckTable(-1)
}

func TestMapTable(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var manga luaManga
	_, err := mapTable(luaTable(t, L, `{
		title = "T", id = 123, url = "u", unknown = "ignored",
		metadata = {
			community_score = "8.5", chapter_count = 12.7, summary = true,
			synonyms = {"a", "b"}, tag_list = "one", genre_list = {},
		},
	}`), &manga)
	if err != nil {
		t.Fatal(err)
	}
	want := luaManga{Title: "T", ID: "123", URL: "u", LuaMetadata: &luaMetadata{
		CommunityScore: 8.5, ChapterCount: 12, Summary: "1",
		Synonyms: []string{"a", "b"}, TagList: []string{"one"}, GenreList: []string{},
	}}
	if !reflect.DeepEqual(manga, want) {
		t.Errorf("got %+v %+v, wanted %+v %+v", manga, manga.LuaMetadata, want, want.LuaMetadata)
	}

	var page luaPage
	if _, err := mapTable(luaTable(t, L, `{url = "u", headers = {Referer = "r", N = 1}}`), &page); err != nil {
		t.Fatal(err)
	}
	if page.URL != "u" || !reflect.DeepEqual(page.Headers, map[string]string{"Referer": "r", "N": "1"}) || page.Cookies != nil {
		t.Errorf("got page %+v", page)
	}

	for expr, want := range map[string]float32{
		`{number = "2"}`:    2,
		`{number = " 2.5"}`: 2.5,
		`{number = ""}`:     0,
		`{number = true}`:   1,
	} {
		var volume luaVolume
		if _, err := mapTable(luaTable(t, L, expr), &volume); err != nil || volume.Number != want {
			t.Errorf("%s: got %v, %v", expr, volume.Number, err)
		}
	}

	for name, expr := range map[string]string{
		"table for a string":  `{title = {}}`,
		"word for a number":   `{metadata = {chapter_count = "many"}}`,
		"map for a list":      `{metadata = {synonyms = {x = "a"}}}`,
		"string for metadata": `{metadata = "m"}`,
		"list for metadata":   `{metadata = {"a"}}`,
		"function for title":  `{title = function() end}`,
		"overflow":            `{metadata = {community_score = "1e40"}}`,
	} {
		var manga luaManga
		if _, err := mapTable(luaTable(t, L, expr), &manga); err == nil {
			t.Errorf("%s: wanted an error, got %+v", name, manga)
		}
	}
}

// the cases that changed when gluamapper was replaced, and back
func TestMapTableLoose(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	for _, tc := range []struct {
		name, expr string
		want       luaManga
	}{
		{"keys ignore case", `{Title = "T", URL = "u", ID = 1}`, luaManga{Title: "T", URL: "u", ID: "1"}},
		{"exact key wins", `{TITLE = "T", title = "t"}`, luaManga{Title: "t"}},
		{"holes in a list", `{metadata = {synonyms = {nil, "b"}, tag_list = {"a", nil, "c"}}}`,
			luaManga{LuaMetadata: &luaMetadata{Synonyms: []string{"", "b"}, TagList: []string{"a", "", "c"}}}},
		{"sparse list", `{metadata = {synonyms = {[1] = "a", [3] = "c"}}}`,
			luaManga{LuaMetadata: &luaMetadata{Synonyms: []string{"a", "", "c"}}}},
		{"keys out of the array part", `{metadata = {synonyms = {[5] = "e", [2] = "b", x = "y"}}}`,
			luaManga{LuaMetadata: &luaMetadata{Synonyms: []string{"b", "e"}}}},
		{"mixed list", `{metadata = {synonyms = {"a", "b", x = "y"}}}`,
			luaManga{LuaMetadata: &luaMetadata{Synonyms: []string{"a", "b"}}}},
		{"mixed manga", `{"x", title = "t", metadata = {"y", summary = "s"}}`,
			luaManga{Title: "t", LuaMetadata: &luaMetadata{Summary: "s"}}},
		{"whole number strings", `{metadata = {chapter_count = " 12.0", community_score = "16 "}}`,
			luaManga{LuaMetadata: &luaMetadata{ChapterCount: 12, CommunityScore: 16}}},
		{"hex count", `{metadata = {chapter_count = "0x10"}}`, luaManga{LuaMetadata: &luaMetadata{ChapterCount: 16}}},
		{"numbers and bools", `{title = 12.5, id = false, metadata = {synonyms = {1, true}}}`,
			luaManga{Title: "12.5", ID: "0", LuaMetadata: &luaMetadata{Synonyms: []string{"1", "1"}}}},
		{"empty metadata", `{metadata = {}}`, luaManga{LuaMetadata: &luaMetadata{}}},
		{"shared table", `(function() local s = {"a"}; return {metadata = {synonyms = s, tag_list = s}} end)()`,
			luaManga{LuaMetadata: &luaMetadata{Synonyms: []string{"a"}, TagList: []string{"a"}}}},
		{"cycle in an unused key", `(function() local m = {title = "t"}; m.self = m; return m end)()`,
			luaManga{Title: "t"}},
	} {
		var manga luaManga
		if _, err := mapTable(luaTable(t, L, tc.expr), &manga); err != nil {
			t.Errorf("%s: %v", tc.name, err)
		} else if !reflect.DeepEqual(manga, tc.want) {
			t.Errorf("%s: got %+v %+v, wanted %+v %+v", tc.name, manga, manga.LuaMetadata, tc.want, tc.want.LuaMetadata)
		}
	}
}

func TestMapTableCyclic(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	for _, expr := range []string{
		`(function() local m = {title = "t"}; m.metadata = m; return m end)()`,
		`(function() local m = {}; m.metadata = {synonyms = {m}}; return m end)()`,
		`(function() local s = {}; s[1] = s; return {metadata = {synonyms = s}} end)()`,
	} {
		var manga luaManga
		if _, err := mapTable(luaTable(t, L, expr), &manga); !errors.Is(err, errCyclic) {
			t.Errorf("%s: wanted a cyclic error, got %v", expr, err)
		}
	}
}

func TestMapTableAllErrors(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var manga luaManga
	_, err := mapTable(luaTable(t, L, `{title = {}, metadata = {chapter_count = "x", synonyms = {x = 1}}}`), &manga)
	if err == nil {
		t.Fatal("wanted an error")
	}
	for _, key := range []string{"'title'", "'metadata.chapter_count'", "'metadata.synonyms[0]'"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q doesn't name %s", err, key)
		}
	}
	if strings.Contains(err.Error(), "\n") {
		t.Errorf("error %q has several lines", err)
	}
}

// every field kind of the tables a script returns
func TestMapTableFields(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	var manga luaManga
	_, err := mapTable(luaTable(t, L, `{
		title = "t", url = "u", id = "i", cover = "c", banner = "b",
		metadata = {
			english_title = "e", romaji_title = "r", native_title = "n", synonyms = {"s"},
			community_score = 7.5, summary = "su", cover_image = "ci", banner_image = "bi",
			tag_list = {"t"}, genre_list = {"g"}, character_list = {"ch"}, author_list = {"a"},
			artist_list = {"ar"}, translator_list = {"tr"}, letterer_list = {"l"},
			date_start = "2020-01-02", date_end = "2021-01-02", provider_publisher = "p",
			publication_status = "FINISHED", publication_format = "MANGA", country_of_origin = "JP",
			chapter_count = 3, extra_notes = "x", source_url = "su", provider_id = "pi",
			provider_id_code = "pc",
		},
	}`), &manga)
	if err != nil {
		t.Fatal(err)
	}
	wantManga := luaManga{Title: "t", URL: "u", ID: "i", Cover: "c", Banner: "b", LuaMetadata: &luaMetadata{
		EnglishTitle: "e", RomajiTitle: "r", NativeTitle: "n", Synonyms: []string{"s"},
		CommunityScore: 7.5, Summary: "su", CoverImage: "ci", BannerImage: "bi",
		TagList: []string{"t"}, GenreList: []string{"g"}, CharacterList: []string{"ch"}, AuthorList: []string{"a"},
		ArtistList: []string{"ar"}, TranslatorList: []string{"tr"}, LettererList: []string{"l"},
		DateStart: "2020-01-02", DateEnd: "2021-01-02", ProviderPublisher: "p",
		PublicationStatus: "FINISHED", PublicationFormat: "MANGA", CountryOfOrigin: "JP",
		ChapterCount: 3, ExtraNotes: "x", SourceURL: "su", ProviderID: "pi", ProviderIDCode: "pc",
	}}
	if !reflect.DeepEqual(manga, wantManga) {
		t.Errorf("manga: got %+v %+v", manga, manga.LuaMetadata)
	}

	var volume luaVolume
	if _, err := mapTable(luaTable(t, L, `{number = 3}`), &volume); err != nil || volume.Number != 3 {
		t.Errorf("volume: got %+v, %v", volume, err)
	}

	var chapter luaChapter
	_, err = mapTable(luaTable(t, L, `{title = "t", url = "u", number = 1.5, date = "2020-01-02", scanlation_group = "g", unavailable = "external"}`), &chapter)
	if err != nil || chapter.Title != "t" || chapter.URL != "u" || chapter.Number != 1.5 || chapter.Date != "2020-01-02" ||
		chapter.ScanlationGroup != "g" || chapter.Unavailable != "external" {
		t.Errorf("chapter: got %+v, %v", chapter, err)
	}

	var page luaPage
	_, err = mapTable(luaTable(t, L, `{url = "u", extension = ".png", headers = {A = "1"}, cookies = {b = 2}}`), &page)
	if err != nil || page.URL != "u" || page.Ext != ".png" ||
		!reflect.DeepEqual(page.Headers, map[string]string{"A": "1"}) || !reflect.DeepEqual(page.Cookies, map[string]string{"b": "2"}) {
		t.Errorf("page: got %+v, %v", page, err)
	}
}

func TestDecodeLogsUnusedKeys(t *testing.T) {
	p := loadTestProvider(t, `
function SearchMangas(query) return {{id = "1", title = "t", metadata = {summery = "typo", score = 1}}} end
function MangaVolumes(manga) return {{number = 1, label = "x"}} end
function VolumeChapters(volume)
	return {{Title = "One", Number = 3, titel = "typo", id = "kept for the script"}, {title = "Extra", titel = "typo", uri = "typo"}}
end
function ChapterPages(chapter) return {} end
`)
	var msgs []string
	l := logger.NewLogger()
	l.SetOnLog(func(format string, a ...any) { msgs = append(msgs, fmt.Sprintf(format, a...)) })
	p.SetLogger(l)
	ctx := context.Background()
	mangas, err := p.SearchMangas(ctx, "q")
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
	// Number counts as number, so Extra goes after it
	if n := chapters[0].Info().Number; n != 3 {
		t.Errorf("chapter Number = 3 got %v", n)
	}
	if n := chapters[1].Info().Number; n != 3.5 {
		t.Errorf("extra chapter got %v, wanted 3.5", n)
	}

	var unused []string
	for _, msg := range msgs {
		if strings.Contains(msg, "isn't used") {
			unused = append(unused, msg)
		}
	}
	// keys kept for the script (label, id, score) aren't typos
	want := []string{
		`test: manga key "metadata.summery" isn't used by mangal, did you mean "summary"?`,
		`test: chapter key "titel" isn't used by mangal, did you mean "title"?`,
		`test: chapter key "uri" isn't used by mangal, did you mean "url"?`,
	}
	if !slices.Equal(unused, want) {
		t.Errorf("logged %q, wanted %q", unused, want)
	}
}

func TestLikelyTypo(t *testing.T) {
	fields := luaFields(reflect.TypeFor[luaManga]())
	for key, want := range map[string]string{
		"titel":             "title",
		"Titl":              "title",
		"uri":               "url",
		"metadata.summery":  "summary",
		"metadata.tag_lsit": "tag_list",
		"ids":               "",
		"label":             "",
		"data":              "",
		"summery":           "", // not a top level field
		"metadata.title":    "",
		"metadata.score":    "",
	} {
		if got, _ := likelyTypo(key, fields); got != want {
			t.Errorf("%q: got %q, want %q", key, got, want)
		}
	}
	// saturno's chapters keep their id for the script
	if got, typo := likelyTypo("id", luaFields(reflect.TypeFor[luaChapter]())); typo {
		t.Errorf("chapter id: got %q", got)
	}
}
