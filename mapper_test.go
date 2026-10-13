package luaprovider

import (
	"reflect"
	"testing"

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
	err := mapTable(luaTable(t, L, `{
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
	if err := mapTable(luaTable(t, L, `{url = "u", headers = {Referer = "r", N = 1}}`), &page); err != nil {
		t.Fatal(err)
	}
	if page.URL != "u" || !reflect.DeepEqual(page.Headers, map[string]string{"Referer": "r", "N": "1"}) || page.Cookies != nil {
		t.Errorf("got page %+v", page)
	}

	var volume luaVolume
	if err := mapTable(luaTable(t, L, `{number = "2"}`), &volume); err != nil || volume.Number != 2 {
		t.Errorf("numeric string: got %v, %v", volume.Number, err)
	}

	for name, expr := range map[string]string{
		"table for a string":  `{title = {}}`,
		"word for a number":   `{metadata = {chapter_count = "many"}}`,
		"map for a list":      `{metadata = {synonyms = {x = "a"}}}`,
		"string for metadata": `{metadata = "m"}`,
	} {
		var manga luaManga
		if err := mapTable(luaTable(t, L, expr), &manga); err == nil {
			t.Errorf("%s: wanted an error, got %+v", name, manga)
		}
	}
}
