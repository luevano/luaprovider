package encoding

import (
	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/lib/encoding/base64"
	"github.com/luevano/luaprovider/lib/encoding/json"
	lua "github.com/yuin/gopher-lua"
)

func Lib(L *lua.LState) *luadoc.Lib {
	return &luadoc.Lib{
		Name:        "encoding",
		Description: "",
		Libs: []*luadoc.Lib{
			base64.Lib(L),
			json.Lib(),
		},
	}
}
