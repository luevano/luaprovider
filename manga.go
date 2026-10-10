package luaprovider

import (
	"encoding/json"
	"time"

	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	lua "github.com/yuin/gopher-lua"
)

var _ mangadata.Manga = (*luaManga)(nil)

type luaManga struct {
	Title  string `gluamapper:"title"`
	URL    string `gluamapper:"url"`
	ID     string `gluamapper:"id"`
	Cover  string `gluamapper:"cover"`
	Banner string `gluamapper:"banner"`
	// LuaMetadata: Optional, see luaMetadata.
	LuaMetadata *luaMetadata `gluamapper:"metadata"`

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
		Title:  m.Title,
		URL:    m.URL,
		ID:     m.ID,
		Cover:  m.Cover,
		Banner: m.Banner,
	}
}

func (m *luaManga) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Info())
}

func (m *luaManga) Metadata() metadata.Metadata {
	return *m.metadata
}

func (m *luaManga) SetMetadata(metadata metadata.Metadata) {
	*m.metadata = metadata
}

// luaMetadata: Manga metadata a provider can return, same names as the libmangal metadata json.
type luaMetadata struct {
	EnglishTitle      string   `gluamapper:"english_title"`
	RomajiTitle       string   `gluamapper:"romaji_title"`
	NativeTitle       string   `gluamapper:"native_title"`
	Synonyms          []string `gluamapper:"synonyms"`
	CommunityScore    float32  `gluamapper:"community_score"`
	Summary           string   `gluamapper:"summary"`
	CoverImage        string   `gluamapper:"cover_image"`
	BannerImage       string   `gluamapper:"banner_image"`
	TagList           []string `gluamapper:"tag_list"`
	GenreList         []string `gluamapper:"genre_list"`
	CharacterList     []string `gluamapper:"character_list"`
	AuthorList        []string `gluamapper:"author_list"`
	ArtistList        []string `gluamapper:"artist_list"`
	TranslatorList    []string `gluamapper:"translator_list"`
	LettererList      []string `gluamapper:"letterer_list"`
	DateStart         string   `gluamapper:"date_start"` // "YYYY-MM-DD"
	DateEnd           string   `gluamapper:"date_end"`   // "YYYY-MM-DD"
	ProviderPublisher string   `gluamapper:"provider_publisher"`
	PublicationStatus string   `gluamapper:"publication_status"` // e.g. "RELEASING", "FINISHED"
	PublicationFormat string   `gluamapper:"publication_format"`
	CountryOfOrigin   string   `gluamapper:"country_of_origin"`
	ChapterCount      int      `gluamapper:"chapter_count"`
	ExtraNotes        string   `gluamapper:"extra_notes"`
	SourceURL         string   `gluamapper:"source_url"`
	ProviderID        string   `gluamapper:"provider_id"`
	ProviderIDCode    string   `gluamapper:"provider_id_code"`
}

// newMetadata builds the manga metadata, from its metadata table if the provider returned one.
//
// The title, images, url and id default to the manga's own.
func (m *luaManga) newMetadata(providerID string) *mangadata.Metadata {
	meta := &mangadata.Metadata{
		EnglishTitle:   m.Title,
		CoverImage:     m.Cover,
		BannerImage:    m.Banner,
		SourceURL:      m.URL,
		ProviderID:     m.ID,
		ProviderIDCode: providerID,
	}
	lm := m.LuaMetadata
	if lm == nil {
		return meta
	}

	if lm.EnglishTitle != "" || lm.RomajiTitle != "" || lm.NativeTitle != "" {
		meta.EnglishTitle, meta.RomajiTitle, meta.NativeTitle = lm.EnglishTitle, lm.RomajiTitle, lm.NativeTitle
	}
	meta.Synonyms = lm.Synonyms
	meta.CommunityScore = lm.CommunityScore
	meta.Summary = lm.Summary
	meta.TagList = lm.TagList
	meta.GenreList = lm.GenreList
	meta.CharacterList = lm.CharacterList
	meta.AuthorList = lm.AuthorList
	meta.ArtistList = lm.ArtistList
	meta.TranslatorList = lm.TranslatorList
	meta.LettererList = lm.LettererList
	meta.DateStart = parseMetadataDate(lm.DateStart)
	meta.DateEnd = parseMetadataDate(lm.DateEnd)
	meta.ProviderPublisher = lm.ProviderPublisher
	meta.PublicationStatus = metadata.Status(lm.PublicationStatus)
	meta.PublicationFormat = lm.PublicationFormat
	meta.CountryOfOrigin = lm.CountryOfOrigin
	meta.ChapterCount = lm.ChapterCount
	meta.ExtraNotes = lm.ExtraNotes
	for field, value := range map[*string]string{
		&meta.CoverImage:     lm.CoverImage,
		&meta.BannerImage:    lm.BannerImage,
		&meta.SourceURL:      lm.SourceURL,
		&meta.ProviderID:     lm.ProviderID,
		&meta.ProviderIDCode: lm.ProviderIDCode,
	} {
		if value != "" {
			*field = value
		}
	}
	return meta
}

// parseMetadataDate parses a "YYYY-MM-DD" date, a zero date if it's empty or invalid.
func parseMetadataDate(date string) metadata.Date {
	parsed, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return metadata.Date{}
	}
	return metadata.Date{Year: parsed.Year(), Month: int(parsed.Month()), Day: parsed.Day()}
}
