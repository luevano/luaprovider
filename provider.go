package luaprovider

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/luevano/libmangal"
	"github.com/luevano/libmangal/logger"
	"github.com/luevano/libmangal/mangadata"
	"github.com/luevano/libmangal/metadata"
	"github.com/luevano/libmangal/netutil"
	luahttp "github.com/luevano/luaprovider/lib/http"
	"github.com/luevano/luaprovider/util"
	"github.com/samber/lo"
	lua "github.com/yuin/gopher-lua"
)

var _ libmangal.Provider = (*provider)(nil)

type provider struct {
	info      libmangal.ProviderInfo
	options   Options
	log       *logger.ProviderLogger
	resources *resources

	// a lua state isn't safe for concurrent use, calls take turns (a 1-slot semaphore, so a
	// waiting call can give up when its ctx is done)
	sem    chan struct{}
	state  *lua.LState
	closed bool
	// cancels the running call when closing, so Close doesn't wait for it
	closing   context.Context
	stopCalls context.CancelFunc
	closeOnce sync.Once
	closeErr  error

	fnSearchMangas,
	fnMangaVolumes,
	fnVolumeChapters,
	fnChapterPages *lua.LFunction
	// nil if the script doesn't define it
	fnMangaByID *lua.LFunction

	unusedKeys sync.Map // logged once, see decode
}

var _ libmangal.MangaByIDProvider = mangaByIDProvider{}

// mangaByIDProvider is a provider whose script defines MangaByID.
type mangaByIDProvider struct {
	*provider
}

// MangaByID calls the script's MangaByID(id), which returns the manga or nil if there's none.
// A not found http error (response:check()) is no manga too.
func (p mangaByIDProvider) MangaByID(ctx context.Context, id string) (mangadata.Manga, bool, error) {
	p.log.Log("getting manga with id %q", id)
	manga, found, err := p.mangaByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: manga %q: %w", p.info.ID, id, err)
	}
	return manga, found, nil
}

func (p mangaByIDProvider) mangaByID(ctx context.Context, id string) (mangadata.Manga, bool, error) {
	ctx, unlock, err := p.lock(ctx)
	if err != nil {
		return nil, false, err
	}
	defer unlock()

	top := p.state.GetTop()
	defer p.state.SetTop(top)

	p.state.SetContext(ctx)
	err = p.state.CallByParam(lua.P{
		Fn:      p.fnMangaByID,
		NRet:    1,
		Protect: true,
	}, lua.LString(id))
	if err != nil {
		return nil, false, callError(ctx, err)
	}

	switch output := p.state.Get(-1).(type) {
	case *lua.LNilType:
		return nil, false, nil
	case *lua.LTable:
		manga, err := p.toManga(output)
		if err != nil {
			return nil, false, err
		}
		return manga, true, nil
	default:
		return nil, false, fmt.Errorf("expected a table or nil, got %s", output.Type())
	}
}

func (p *provider) String() string {
	return p.info.Name
}

// callError is the error of a lua call: ctx's if it's done (lua only has its message), else
// the Go error the script raised, if any.
func callError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return util.GoError(err)
}

// ErrClosed is returned by the calls to a closed provider.
var ErrClosed = errors.New("provider closed")

// lock takes the lua state for a call, with a ctx that's also cancelled by Close. Call unlock
// when done.
func (p *provider) lock(ctx context.Context) (_ context.Context, unlock func(), _ error) {
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	if p.closed {
		<-p.sem
		return nil, nil, ErrClosed
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(p.closing, cancel)
	return ctx, func() {
		stop()
		cancel()
		<-p.sem
	}, nil
}

func (p *provider) Close() error {
	p.closeOnce.Do(func() {
		p.stopCalls()
		p.sem <- struct{}{}
		defer func() { <-p.sem }()
		p.closed = true
		p.closeErr = p.resources.close()
		p.state.Close()
	})
	return p.closeErr
}

func (p *provider) Info() libmangal.ProviderInfo {
	return p.info
}

func (p *provider) SetLogger(logger *logger.Logger) {
	p.log.SetLogger(logger)
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
	ctx, unlock, err := p.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()

	// the returned value has to be popped, else the stack grows until the state breaks
	top := p.state.GetTop()
	defer p.state.SetTop(top)

	p.state.SetContext(ctx)
	err = p.state.CallByParam(lua.P{
		Fn:      lfunc,
		NRet:    1,
		Protect: true,
	}, lo.Map(args, func(arg Input, _ int) lua.LValue {
		return arg.IntoLValue()
	})...)
	if err != nil {
		return nil, callError(ctx, err)
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
			return nil, fmt.Errorf("item %d: expected a table, got %s", i+1, value.Type())
		}

		item, err := convert(i, table)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i+1, err)
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

func (p *provider) SearchMangas(
	ctx context.Context,
	query string,
) ([]mangadata.Manga, error) {
	p.log.Log("searching mangas with query %q", query)

	mangas, err := loadItems(
		ctx,
		p,
		p.fnSearchMangas,
		func(_ int, table *lua.LTable) (mangadata.Manga, error) {
			return p.toManga(table)
		},
		luaString(query),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: search %q: %w", p.info.ID, query, err)
	}
	return mangas, nil
}

// decode maps a table the script returned (what: "manga", "volume"...) into dst and logs the keys
// mangal doesn't use that look like typos of a field, once each. Other keys are the script's
// own (the tables are passed back to it).
func (p *provider) decode(what string, table *lua.LTable, dst any) (mapstructure.Metadata, error) {
	md, err := mapTable(table, dst)
	slices.Sort(md.Unused)
	var fields []string
	if len(md.Unused) > 0 {
		fields = luaFields(reflect.TypeOf(dst))
	}
	for _, key := range md.Unused {
		field, typo := likelyTypo(key, fields)
		if !typo {
			continue
		}
		if _, logged := p.unusedKeys.LoadOrStore(what+"."+key, true); !logged {
			p.log.Log("%s key %q isn't used by mangal, did you mean %q?", what, key, field)
		}
	}
	return md, err
}

// toManga converts a manga table the script returned.
func (p *provider) toManga(table *lua.LTable) (*luaManga, error) {
	var manga luaManga
	if _, err := p.decode("manga", table, &manga); err != nil {
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
}

func (p *provider) MangaVolumes(
	ctx context.Context,
	manga mangadata.Manga,
) ([]mangadata.Volume, error) {
	m, ok := manga.(*luaManga)
	if !ok {
		return nil, fmt.Errorf("expected a luaprovider manga, got %T", manga)
	}

	p.log.Log("fetching volumes for manga %q", m)
	volumes, err := loadItems(
		ctx,
		p,
		p.fnMangaVolumes,
		func(_ int, table *lua.LTable) (mangadata.Volume, error) {
			var volume luaVolume
			if _, err := p.decode("volume", table, &volume); err != nil {
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

	p.log.Log("fetching chapters for volume %s", v)
	unnumbered := map[int]bool{}
	chapters, err := loadItems(
		ctx,
		p,
		p.fnVolumeChapters,
		func(i int, table *lua.LTable) (mangadata.Chapter, error) {
			var chapter luaChapter
			md, err := p.decode("chapter", table, &chapter)
			if err != nil {
				return nil, err
			}
			if chapter.Title == "" {
				return nil, errors.New("missing title")
			}
			unnumbered[i] = slices.Contains(md.Unset, "number")

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
	// chapters without a number (e.g. "Extra") go right after the previous one, on a number
	// no other chapter has
	taken := map[int64]bool{}
	for i, c := range chapters {
		if !unnumbered[i] {
			taken[scaledNumber(c.(*luaChapter).Number)] = true
		}
	}
	var last float32
	for i, c := range chapters {
		chapter := c.(*luaChapter)
		if unnumbered[i] {
			chapter.Number = nextExtraNumber(last, taken)
			taken[scaledNumber(chapter.Number)] = true
		}
		last = chapter.Number
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

	p.log.Log("fetching pages for chapter %q", c)
	pages, err := loadItems(
		ctx,
		p,
		p.fnChapterPages,
		func(_ int, table *lua.LTable) (mangadata.Page, error) {
			var page luaPage
			if _, err := p.decode("page", table, &page); err != nil {
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
			p.log.Log("invalid chapter date %q, using today: %s", date, err)
			parsed = time.Now()
		}
	}
	return metadata.Date{Year: parsed.Year(), Month: int(parsed.Month()), Day: parsed.Day()}
}

// numberScale: Chapter numbers are compared and stepped as integers in millionths, float32
// sums drift (10.5+0.1 isn't 10.6).
const numberScale = 1_000_000

func scaledNumber(n float32) int64 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(float64(n), 'f', -1, 32), 64)
	return int64(math.Round(f * numberScale))
}

func unscaledNumber(n int64) float32 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(float64(n)/numberScale, 'f', -1, 64), 32)
	return float32(f)
}

// nextExtraNumber returns the number for a chapter without one that comes right after last
// and isn't taken: after 5 it's 5.5, after 5.5 (or if 5.5 is taken) 5.6, ..., then 5.91,
// 5.92, ... It stays below the next whole number a chapter has; when no chapter has 6, a long
// run of extras goes on past it (6.5, 6.6, ...), never on the whole number itself.
func nextExtraNumber(last float32, taken map[int64]bool) float32 {
	// past a whole number only when no chapter has it
	for base := scaledNumber(last); ; base = (base/numberScale + 1) * numberScale {
		next := (base/numberScale + 1) * numberScale
		n := base
		if n%numberScale == 0 {
			n += numberScale / 2
			if n < next && !taken[n] {
				return unscaledNumber(n)
			}
		}
		for step := int64(numberScale / 10); step > 0; {
			if n+step >= next {
				step /= 10
				continue
			}
			n += step
			// float32 can't hold every millionth of big numbers (it'd round to a taken or whole one)
			if !taken[n] && scaledNumber(unscaledNumber(n)) == n {
				return unscaledNumber(n)
			}
		}
		if taken[next] {
			break
		}
	}
	// shortcut: about 40 extras in a row fit before a whole number a chapter has (float32),
	// later ones share the previous extra's number instead of taking a real chapter's; backlog
	// "extra chapters"
	return last
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

// pageRetries is how many times a page is tried again after a 429, waiting Retry-After
// (at most maxRetryWait) or pageRetryWait, doubled each time.
const (
	pageRetries  = libmangal.DefaultPageRetries
	maxRetryWait = libmangal.DefaultMaxPageRetryWait
)

var pageRetryWait = libmangal.DefaultPageRetryWait

// getPageImage gets the page, an *HTTPError or *RateLimitError for an error status.
func (p *provider) getPageImage(
	ctx context.Context,
	page luaPage,
) ([]byte, error) {
	image, err := libmangal.FetchPageImage(ctx, p.options.HTTPClient, libmangal.PageImageRequest{
		URL:     page.URL,
		Headers: page.Headers,
		Cookies: page.Cookies,
	}, libmangal.FetchPageImageOptions{
		Retries:      pageRetries,
		RetryWait:    pageRetryWait,
		MaxRetryWait: maxRetryWait,
		OnRetry: func(retry int, wait time.Duration) {
			p.log.Status(ctx, "page of chapter %q: 429 too many requests, retry %d/%d in %s", page.chapter, retry, pageRetries, wait.Round(time.Second))
		},
		StatusError: luahttp.NewError,
	})
	// the same error as the scripts' requests
	if rateLimit, ok := errors.AsType[*netutil.RateLimitError](err); ok {
		return nil, &RateLimitError{Method: http.MethodGet, URL: page.URL, RetryAfter: rateLimit.RetryAfter}
	}
	return image, err
}
