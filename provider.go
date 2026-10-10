package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/samber/lo"
	"github.com/yuin/gluamapper"
	lua "github.com/yuin/gopher-lua"
)

var _ libmangal.Provider = (*provider)(nil)

type provider struct {
	info      libmangal.ProviderInfo
	options   Options
	log       *providerLogger
	resources *resources

	// a lua state isn't safe for concurrent use, calls take turns
	mu    sync.Mutex
	state *lua.LState
	// closing a lua state twice panics
	closeOnce sync.Once
	closeErr  error

	fnSearchMangas,
	fnMangaVolumes,
	fnVolumeChapters,
	fnChapterPages *lua.LFunction
}

func (p *provider) String() string {
	return p.info.Name
}

func (p *provider) Close() error {
	p.closeOnce.Do(func() {
		p.closeErr = p.resources.close()
		p.state.Close()
	})
	return p.closeErr
}

func (p *provider) Info() libmangal.ProviderInfo {
	return p.info
}

func (p *provider) SetLogger(logger *logger.Logger) {
	p.log.setLogger(logger)
}

type intoLValue interface {
	IntoLValue() lua.LValue
}

// loadItems runs the lua function and converts each item of the table it returns.
func loadItems[Input intoLValue, Output any](
	ctx context.Context,
	p *provider,
	lfunc *lua.LFunction,
	convert func(int, *lua.LTable) (Output, error),
	args ...Input,
) ([]Output, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// the returned value has to be popped, else the stack grows until the state breaks
	top := p.state.GetTop()
	defer p.state.SetTop(top)

	p.state.SetContext(ctx)
	err := p.state.CallByParam(lua.P{
		Fn:      lfunc,
		NRet:    1,
		Protect: true,
	}, lo.Map(args, func(arg Input, _ int) lua.LValue {
		return arg.IntoLValue()
	})...)
	if err != nil {
		return nil, err
	}

	output, ok := p.state.Get(-1).(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("expected a table, got %s", p.state.Get(-1).Type())
	}

	var values []lua.LValue
	output.ForEach(func(_ lua.LValue, value lua.LValue) {
		values = append(values, value)
	})

	items := make([]Output, len(values))
	for i, value := range values {
		table, ok := value.(*lua.LTable)
		if !ok {
			return nil, fmt.Errorf("item %d: expected a table, got %s", i, value.Type())
		}

		item, err := convert(i, table)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		items[i] = item
	}
	return items, nil
}

// luaString wraps string to make IntoLValue method available.
// Required by the loadItems function.
type luaString string

func (l luaString) IntoLValue() lua.LValue {
	return lua.LString(l)
}

var (
	glOpt    gluamapper.Option = gluamapper.Option{NameFunc: gluamapper.Id, TagName: "gluamapper", ErrorUnused: false}
	glMapper gluamapper.Mapper = *gluamapper.NewMapper(glOpt)
)

func (p *provider) SearchMangas(
	ctx context.Context,
	query string,
) ([]mangadata.Manga, error) {
	p.log.log("searching mangas with query %q", query)

	mangas, err := loadItems(
		ctx,
		p,
		p.fnSearchMangas,
		func(i int, table *lua.LTable) (mangadata.Manga, error) {
			var manga luaManga
			if err := glMapper.Map(table, &manga); err != nil {
				return nil, err
			}
			if manga.ID == "" {
				return nil, errors.New("missing id")
			}
			if manga.Title == "" {
				return nil, errors.New("missing title")
			}

			var meta metadata.Metadata = manga.newMetadata(p.info.ID)
			manga.metadata = &meta
			manga.table = table
			return &manga, nil
		},
		luaString(query),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: search %q: %w", p.info.ID, query, err)
	}
	return mangas, nil
}

func (p *provider) MangaVolumes(
	ctx context.Context,
	manga mangadata.Manga,
) ([]mangadata.Volume, error) {
	m, ok := manga.(*luaManga)
	if !ok {
		return nil, fmt.Errorf("expected a luaprovider manga, got %T", manga)
	}

	p.log.log("fetching volumes for manga %q", m)
	volumes, err := loadItems(
		ctx,
		p,
		p.fnMangaVolumes,
		func(_ int, table *lua.LTable) (mangadata.Volume, error) {
			var volume luaVolume
			if err := glMapper.Map(table, &volume); err != nil {
				return nil, err
			}

			// gives the volume access to its manga
			table.RawSetString("manga", m.table)
			volume.table = table
			volume.manga = m
			return &volume, nil
		},
		m,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: volumes of %q: %w", p.info.ID, m, err)
	}
	return volumes, nil
}

func (p *provider) VolumeChapters(
	ctx context.Context,
	volume mangadata.Volume,
) ([]mangadata.Chapter, error) {
	v, ok := volume.(*luaVolume)
	if !ok {
		return nil, fmt.Errorf("expected a luaprovider volume, got %T", volume)
	}

	p.log.log("fetching chapters for volume %s", v)
	var last float32
	chapters, err := loadItems(
		ctx,
		p,
		p.fnVolumeChapters,
		func(_ int, table *lua.LTable) (mangadata.Chapter, error) {
			var chapter luaChapter
			if err := glMapper.Map(table, &chapter); err != nil {
				return nil, err
			}
			if chapter.Title == "" {
				return nil, errors.New("missing title")
			}
			// chapters without a number (e.g. "Extra") go right after the previous one
			if table.RawGetString("number") == lua.LNil {
				chapter.Number = nextExtraNumber(last)
			}
			last = chapter.Number

			// gives the chapter access to its volume
			table.RawSetString("volume", v.table)
			chapter.date = p.parseDate(chapter.Date)
			chapter.table = table
			chapter.volume = v
			return &chapter, nil
		},
		v,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: chapters of %q vol %s: %w", p.info.ID, v.manga, v, err)
	}
	if !p.options.Filter.ShowUnavailableChapters {
		chapters = slices.DeleteFunc(chapters, func(c mangadata.Chapter) bool {
			return c.Info().Unavailable != ""
		})
	}
	return chapters, nil
}

func (p *provider) ChapterPages(
	ctx context.Context,
	chapter mangadata.Chapter,
) ([]mangadata.Page, error) {
	c, ok := chapter.(*luaChapter)
	if !ok {
		return nil, fmt.Errorf("expected a luaprovider chapter, got %T", chapter)
	}

	p.log.log("fetching pages for chapter %q", c)
	pages, err := loadItems(
		ctx,
		p,
		p.fnChapterPages,
		func(_ int, table *lua.LTable) (mangadata.Page, error) {
			var page luaPage
			if err := glMapper.Map(table, &page); err != nil {
				return nil, err
			}
			page.chapter = c

			if page.URL == "" {
				return nil, errors.New("missing url")
			}
			if page.Ext == "" {
				page.Ext = ".jpg"
			}
			if !fileExtensionRegex.MatchString(page.Ext) {
				return nil, fmt.Errorf("invalid image extension %q", page.Ext)
			}

			if page.Headers == nil {
				page.Headers = make(map[string]string)
			}
			for key, value := range map[string]string{
				"Referer":    c.URL,
				"Accept":     "image/webp,image/apng,image/*,*/*;q=0.8",
				"User-Agent": p.options.UserAgent,
			} {
				if _, ok := page.Headers[key]; !ok {
					page.Headers[key] = value
				}
			}
			return &page, nil
		},
		c,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: pages of chapter %q: %w", p.info.ID, c, err)
	}
	return pages, nil
}

// parseDate parses a "YYYY-MM-DD" date, today if it's empty or invalid.
func (p *provider) parseDate(date string) metadata.Date {
	parsed := time.Now()
	if date != "" {
		var err error
		if parsed, err = time.Parse(time.DateOnly, date); err != nil {
			p.log.log("invalid chapter date %q, using today: %s", date, err)
			parsed = time.Now()
		}
	}
	return metadata.Date{Year: parsed.Year(), Month: int(parsed.Month()), Day: parsed.Day()}
}

// nextExtraNumber returns the number for a chapter without one that comes right after last,
// e.g. an "Extra" after chapter 5 is 5.5, after 5.5 it's 5.6.
func nextExtraNumber(last float32) float32 {
	// avoids floating point precision issues https://stackoverflow.com/a/56300186
	if last == float32(int(last)) {
		return float32((float64(last*10) + 5) / 10)
	}
	return float32((float64(last*10) + 1) / 10)
}

func (p *provider) GetPageImage(
	ctx context.Context,
	page mangadata.Page,
) ([]byte, error) {
	page_, ok := page.(*luaPage)
	if !ok {
		return nil, fmt.Errorf("expected a luaprovider page, got %T", page)
	}

	image, err := p.getPageImage(ctx, *page_)
	if err != nil {
		return nil, fmt.Errorf("%s: image for chapter %q: %w", p.info.ID, page_.chapter, err)
	}
	return image, nil
}

func (p *provider) getPageImage(
	ctx context.Context,
	page luaPage,
) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, page.URL, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range page.Headers {
		request.Header.Set(key, value)
	}

	for key, value := range page.Cookies {
		request.AddCookie(&http.Cookie{Name: key, Value: value})
	}

	response, err := p.options.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %d %s", page.URL, response.StatusCode, strings.ToLower(http.StatusText(response.StatusCode)))
	}

	return io.ReadAll(response.Body)
}
