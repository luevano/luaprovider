package luaprovider

import (
	"encoding/json"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	lua "github.com/yuin/gopher-lua"
)

var _ mangadata.Chapter = (*luaChapter)(nil)

type luaChapter struct {
	Title           string  `gluamapper:"title"`
	URL             string  `gluamapper:"url"`
	Number          float32 `gluamapper:"number"`
	Date            string  `gluamapper:"date"` // fmt "YYYY-MM-dd" or empty
	ScanlationGroup string  `gluamapper:"scanlation_group"`

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
	today := time.Now()
	date := metadata.Date{
		Year:  today.Year(),
		Month: int(today.Month()),
		Day:   today.Day(),
	}
	if c.Date != "" {
		parsedDate, err := time.Parse(time.DateOnly, c.Date)
		// TODO: use logger when err
		if err == nil {
			date.Year = parsedDate.Year()
			date.Month = int(parsedDate.Month())
			date.Day = parsedDate.Day()
		}
	}
	return mangadata.ChapterInfo{
		Title:           c.Title,
		URL:             c.URL,
		Number:          c.Number,
		Date:            date,
		ScanlationGroup: c.ScanlationGroup,
	}
}
