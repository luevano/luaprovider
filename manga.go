package luaprovider

import (
	"encoding/json"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	lua "github.com/yuin/gopher-lua"
)

var _ mangadata.Manga = (*luaManga)(nil)

type luaManga struct {
	Title         string `gluamapper:"title"`
	AnilistSearch string `gluamapper:"anilist_search"`
	URL           string `gluamapper:"url"`
	ID            string `gluamapper:"id"`
	Cover         string `gluamapper:"cover"`
	Banner        string `gluamapper:"banner"`

	metadata *metadata.Metadata
	table    *lua.LTable
}

func (m *luaManga) String() string {
	return m.Title
}

func (m *luaManga) IntoLValue() lua.LValue {
	return m.table
}

func (m *luaManga) Info() mangadata.MangaInfo {
	return mangadata.MangaInfo{
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

func (m *luaManga) Metadata() *metadata.Metadata {
	return m.metadata
}

func (m *luaManga) SetMetadata(metadata *metadata.Metadata) {
	m.metadata = metadata
}
