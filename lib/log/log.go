package log

import (
	luadoc "github.com/luevano/gopher-luadoc"
	lua "github.com/yuin/gopher-lua"
)

const libName = "log"

// Lib: Logs messages of the provider through mangal's logs.
func Lib(log func(format string, a ...any)) *luadoc.Lib {
	return &luadoc.Lib{
		Name:        libName,
		Description: "Logging through mangal's logs, prefixed with the provider id.",
		Funcs: []*luadoc.Func{
			{
				Name:        "print",
				Description: "Log a message.",
				Value: func(L *lua.LState) int {
					log("%s", L.CheckString(1))
					return 0
				},
				Params: []*luadoc.Param{
					{
						Name: "message",
						Type: luadoc.String,
					},
				},
			},
		},
	}
}
