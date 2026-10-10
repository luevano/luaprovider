package options

import (
	luadoc "github.com/luevano/gopher-luadoc"
	lua "github.com/yuin/gopher-lua"
)

const libName = "options"

// Filter: The user's filters, same as mangoprovider's.
type Filter struct {
	// NSFW: Whether the results can include NSFW content.
	NSFW bool
	// Language: Of the mangas/chapters, e.g. "en".
	Language string
	// AvoidDuplicateChapters: Keep only one chapter per number.
	AvoidDuplicateChapters bool
	// ShowUnavailableChapters: Also show chapters that can't be downloaded,
	// else luaprovider drops the ones with "unavailable" set.
	ShowUnavailableChapters bool
}

func Lib(filter Filter) *luadoc.Lib {
	return &luadoc.Lib{
		Name:        libName,
		Description: "The user's options, scripts should respect them when the source supports it.",
		Vars: []*luadoc.Var{
			{
				Name:        "nsfw",
				Description: "Whether the results can include NSFW content.",
				Value:       lua.LBool(filter.NSFW),
			},
			{
				Name:        "language",
				Description: `Language of the mangas/chapters, e.g. "en".`,
				Value:       lua.LString(filter.Language),
			},
			{
				Name:        "avoid_duplicate_chapters",
				Description: "Keep only one chapter per number.",
				Value:       lua.LBool(filter.AvoidDuplicateChapters),
			},
			{
				Name:        "show_unavailable_chapters",
				Description: `Whether chapters that can't be downloaded are shown, those with "unavailable" set are dropped otherwise.`,
				Value:       lua.LBool(filter.ShowUnavailableChapters),
			},
		},
	}
}
