package luaprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	lua "github.com/yuin/gopher-lua"
)

var _ mangadata.Chapter = (*luaChapter)(nil)

type luaChapter struct {
	Title           string  `lua:"title"`
	URL             string  `lua:"url"`
	Number          float32 `lua:"number"`
	Date            string  `lua:"date"` // "YYYY-MM-DD", today if empty
	ScanlationGroup string  `lua:"scanlation_group"`
	// Unavailable: Why it can't be downloaded (e.g. "external"), empty if it can.
	Unavailable string `lua:"unavailable"`

	date   metadata.Date
	volume *luaVolume
	table  *lua.LTable
}

func (c *luaChapter) String() string {
	return c.Title
}

func (c *luaChapter) Volume() mangadata.Volume {
	return c.volume
}

func (c *luaChapter) IntoLValue() lua.LValue {
	return c.table
}

func (c *luaChapter) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Info())
}

func (c *luaChapter) Info() mangadata.ChapterInfo {
	return mangadata.ChapterInfo{
		Title:           c.Title,
		URL:             c.URL,
		Number:          c.Number,
		Date:            c.date,
		ScanlationGroup: c.ScanlationGroup,
		Unavailable:     c.Unavailable,
	}
}
