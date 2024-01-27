package luaprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal"
	lua "github.com/yuin/gopher-lua"
)

var (
	_ libmangal.MangaWithSeriesJSON = (*luaManga)(nil)
	_ libmangal.Manga               = (*luaManga)(nil)
)

type luaManga struct {
	Title         string `gluamapper:"title"`
	AnilistSearch string `gluamapper:"anilist_search"`
	URL           string `gluamapper:"url"`
	ID            string `gluamapper:"id"`
	Cover         string `gluamapper:"cover"`
	Banner        string `gluamapper:"banner"`

	AnilistSet_ bool                   `gluamapper:"-"`
	Anilist_    libmangal.AnilistManga `gluamapper:"-"`
	table       *lua.LTable
}

func (m *luaManga) String() string {
	return m.Title
}

func (m *luaManga) IntoLValue() lua.LValue {
	return m.table
}

func (m *luaManga) Info() libmangal.MangaInfo {
	return libmangal.MangaInfo{
		Title:         m.Title,
		AnilistSearch: m.AnilistSearch,
		URL:           m.URL,
		ID:            m.ID,
		Cover:         m.Cover,
		Banner:        m.Banner,
	}
}

func (m *luaManga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}

func (m *luaManga) AnilistManga() (libmangal.AnilistManga, bool) {
	return m.Anilist_, m.AnilistSet_
}

func (m *luaManga) SetAnilistManga(anilist libmangal.AnilistManga) {
	m.Anilist_ = anilist
	m.AnilistSet_ = true
}

func (m *luaManga) SeriesJSON() (libmangal.SeriesJSON, bool, error) {
	if !m.AnilistSet_ {
		// TODO: once logger is setup, use it
		// Log(fmt.Sprintf("manga %q doesn't contain anilist data", m.Title))
		return libmangal.SeriesJSON{}, false, nil
	}

	return libmangal.AnilistSeriesJSON(m.Anilist_), true, nil
}
