package regexp

import (
	"regexp"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/util"
	"github.com/mvdan/xurls"
	lua "github.com/yuin/gopher-lua"
)

const libName = "regex"

func Lib(L *lua.LState) *luadoc.Lib {
	toLValue := func(r *regexp.Regexp) *lua.LUserData {
		ud := L.NewUserData()
		ud.Value = r
		L.SetMetatable(ud, L.GetTypeMetatable(patternTypeName))
		return ud
	}

	classPattern := &luadoc.Class{
		Name:        patternTypeName,
		Description: "Compiled regular expression",
		Methods: []*luadoc.Method{
			{
				Name: "find_submatch",
				Description: `Returns a slice of strings holding the text of the leftmost match of the regular
expression in s and the matches, if any, of its subexpressions. A return value of empty table indicates no match.`,
				Value: regexpFindSubmatch,
				Params: []*luadoc.Param{
					{
						Name:        "text",
						Description: "A string to search in",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "matches",
						Description: "Found matches",
						Type:        luadoc.Table,
					},
				},
			},
			{
				Name:        "match",
				Description: "Reports whether the string s contains any match of the regular expression.",
				Value:       regexpMatch,
				Params: []*luadoc.Param{
					{
						Name:        "text",
						Description: "A string to search in",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "matched",
						Description: "Whether the string s contains any match of the regular expression.",
						Type:        luadoc.Table,
					},
				},
			},
			{
				Name: "replace_all",
				Description: `Returns a copy of text, replacing matches of the regexp with the replacement string.
Inside replacement, $ signs are interpreted as in expand, so for instance $1 represents the text of the first submatch.`,
				Value: regexpReplaceAll,
				Params: []*luadoc.Param{
					{
						Name:        "text",
						Description: "A string to replace matches in",
						Type:        luadoc.String,
					},
					{
						Name:        "replacement",
						Description: "A string to replace matches with",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "replaced",
						Description: "The result of the replacement",
						Type:        luadoc.String,
					},
				},
			},
			{
				Name:        "split",
				Description: "Splits the given text into a table of strings.",
				Value:       regexpSplit,
				Params: []*luadoc.Param{
					{
						Name:        "text",
						Description: "The text to split",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "parts",
						Description: "The result of the split",
						Type:        luadoc.Table,
					},
				},
			},
			{
				Name:        "groups",
				Description: "Returns a table of all capture groups.",
				Value:       regexpGroups,
				Params: []*luadoc.Param{
					{
						Name:        "text",
						Description: "The text to split",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "groups",
						Description: "A table of all capture groups",
						Type:        luadoc.Table,
					},
				},
			},
			{
				Name:        "replace_all_func",
				Description: "Returns a copy of text, replacing matches of the regexp with the replacement function.",
				Value:       regexpReplaceAllFunc,
				Params: []*luadoc.Param{
					{
						Name:        "text",
						Description: "A string to replace matches in",
						Type:        luadoc.String,
					},
					{
						Name:        "replacer",
						Description: "A function to replace matches with",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "match",
									Type: luadoc.String,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "replaced",
									Type: luadoc.String,
								},
							},
						}.AsType(),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "replaced",
						Description: "The result of the replacement",
						Type:        luadoc.String,
					},
				},
			},
		},
	}

	return &luadoc.Lib{
		Name:        "regexp",
		Description: "A regular expression library.",
		Vars: []*luadoc.Var{
			{
				Name:        "urls_relaxed",
				Description: "Matches all the urls it can find.",
				Value:       toLValue(xurls.Relaxed),
				Type:        patternTypeName,
			},
			{
				Name:        "urls_strict",
				Description: "Only matches urls with a scheme to avoid false positives.",
				Value:       toLValue(xurls.Strict),
				Type:        patternTypeName,
			},
		},
		Funcs: []*luadoc.Func{
			{
				Name:        "match",
				Description: "Matches the given pattern against the given text.",
				Value:       match,
				Params: []*luadoc.Param{
					{
						Name:        "pattern",
						Description: "The pattern to match",
						Type:        luadoc.String,
					},
					{
						Name:        "text",
						Description: "The text to match against",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "matched",
						Description: "A table of all matches",
						Type:        luadoc.Table,
					},
				},
			},
			{
				Name:        "compile",
				Description: "Compiles the given pattern into a regular expression.",
				Value:       compile,
				Params: []*luadoc.Param{
					{
						Name:        "pattern",
						Description: "The pattern to compile",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "regexp",
						Description: "The compiled regular expression",
						Type:        patternTypeName,
					},
				},
			},
		},
		Classes: []*luadoc.Class{
			classPattern,
		},
	}
}

func match(L *lua.LState) int {
	pattern := L.CheckString(1)
	text := L.CheckString(2)
	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	L.Push(lua.LBool(matched))
	return 1
}

func compile(L *lua.LState) int {
	pattern := L.CheckString(1)
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	util.Push(L, compiled, patternTypeName)
	return 1
}
