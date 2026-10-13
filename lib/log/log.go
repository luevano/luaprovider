package log

import (
	"context"

	luadoc "github.com/luevano/gopher-luadoc"
	lua "github.com/yuin/gopher-lua"
)

const libName = "log"

// Lib: Logs messages of the provider through mangal's logs, status shows them to the user too.
func Lib(log func(format string, a ...any), status func(ctx context.Context, message string)) *luadoc.Lib {
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
			{
				Name:        "status",
				Description: `Log a message and show it as what the provider is doing now (e.g. "page 3 of 5", "rate limited, retrying in 10s"), on mangal's status line or spinner.`,
				Value: func(L *lua.LState) int {
					status(L.Context(), L.CheckString(1))
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
