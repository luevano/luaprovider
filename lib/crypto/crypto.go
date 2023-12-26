package crypto

import (
	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/lib/crypto/aes"
	"github.com/luevano/luaprovider/lib/crypto/md5"
	"github.com/luevano/luaprovider/lib/crypto/sha1"
	"github.com/luevano/luaprovider/lib/crypto/sha256"
	"github.com/luevano/luaprovider/lib/crypto/sha512"
	lua "github.com/yuin/gopher-lua"
)

func Lib(L *lua.LState) *luadoc.Lib {
	return &luadoc.Lib{
		Name:        "crypto",
		Description: "Various cryptographic functions.",
		Libs: []*luadoc.Lib{
			aes.Lib(),
			md5.Lib(),
			sha1.Lib(),
			sha256.Lib(),
			sha512.Lib(),
		},
	}
}
