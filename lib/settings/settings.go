package settings

import (
	"strconv"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/libmangal"
	lua "github.com/yuin/gopher-lua"
)

const libName = "settings"

// Lib gives scripts the values of the provider's declared settings
// (libmangal.ProviderInfo.Settings): values has one for every declared key, see
// libmangal.ProviderInfo.SettingValues.
func Lib(declared []libmangal.ProviderSetting, values map[string]string) *luadoc.Lib {
	kinds := make(map[string]libmangal.SettingKind, len(declared))
	for _, s := range declared {
		kinds[s.Key] = s.Kind
	}
	return &luadoc.Lib{
		Name:        libName,
		Description: "The user's values of the settings the provider declares in its mangal.toml ([[settings]]).",
		Funcs: []*luadoc.Func{
			{
				Name:        "get",
				Description: "The value of a declared setting, its default if the user didn't set it. Raises for an undeclared key.",
				Value: func(L *lua.LState) int {
					key := L.CheckString(1)
					kind, ok := kinds[key]
					if !ok {
						L.ArgError(1, "undeclared setting "+strconv.Quote(key))
					}
					L.Push(value(L, kind, values[key]))
					return 1
				},
				Params: []*luadoc.Param{
					{
						Name:        "key",
						Description: "Key of the setting.",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "value",
						Description: "A boolean for bool, a number for int, a list of strings for list, a string for the other kinds (a duration as text, e.g. \"500ms\").",
						Type:        luadoc.Any,
					},
				},
			},
		},
	}
}

// value is the Lua value of a setting's text, already checked against its kind.
func value(L *lua.LState, kind libmangal.SettingKind, text string) lua.LValue {
	switch kind {
	case libmangal.SettingBool:
		b, _ := strconv.ParseBool(text)
		return lua.LBool(b)
	case libmangal.SettingInt:
		n, _ := strconv.ParseInt(text, 10, 64)
		return lua.LNumber(n)
	case libmangal.SettingList:
		list := L.NewTable()
		for _, item := range libmangal.SplitList(text) {
			list.Append(lua.LString(item))
		}
		return list
	default:
		return lua.LString(text)
	}
}
