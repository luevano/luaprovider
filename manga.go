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
	Title  string `lua:"title"`
	URL    string `lua:"url"`
	ID     string `lua:"id"`
	Cover  string `lua:"cover"`
	Banner string `lua:"banner"`
	// LuaMetadata: Optional, see luaMetadata.
	LuaMetadata *luaMetadata `lua:"metadata"`

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
	EnglishTitle      string   `lua:"english_title"`
	RomajiTitle       string   `lua:"romaji_title"`
	NativeTitle       string   `lua:"native_title"`
	Synonyms          []string `lua:"synonyms"`
	CommunityScore    float32  `lua:"community_score"`
	Summary           string   `lua:"summary"`
	CoverImage        string   `lua:"cover_image"`
	BannerImage       string   `lua:"banner_image"`
	TagList           []string `lua:"tag_list"`
	GenreList         []string `lua:"genre_list"`
	CharacterList     []string `lua:"character_list"`
	AuthorList        []string `lua:"author_list"`
	ArtistList        []string `lua:"artist_list"`
	TranslatorList    []string `lua:"translator_list"`
	LettererList      []string `lua:"letterer_list"`
	DateStart         string   `lua:"date_start"` // "YYYY-MM-DD"
	DateEnd           string   `lua:"date_end"`   // "YYYY-MM-DD"
	ProviderPublisher string   `lua:"provider_publisher"`
	PublicationStatus string   `lua:"publication_status"` // e.g. "RELEASING", "FINISHED"
	PublicationFormat string   `lua:"publication_format"`
	CountryOfOrigin   string   `lua:"country_of_origin"`
	ChapterCount      int      `lua:"chapter_count"`
	ExtraNotes        string   `lua:"extra_notes"`
	SourceURL         string   `lua:"source_url"`
	ProviderID        string   `lua:"provider_id"`
	ProviderIDCode    string   `lua:"provider_id_code"`
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
