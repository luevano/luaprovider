package base64

import (
	"encoding/base64"

	luadoc "github.com/luevano/gopher-luadoc"
	lua "github.com/yuin/gopher-lua"
)

const (
	libName          = "base64"
	encodingTypeName = libName + "_encoding"
)

func encodingToLua(L *lua.LState, encoding *base64.Encoding) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = encoding
	return ud
}

func Lib(L *lua.LState) *luadoc.Lib {
	return &luadoc.Lib{
		Name:        libName,
		Description: "Base64 encoding and decoding.",
		Vars: []*luadoc.Var{
			{
				Name:        "std_encoding",
				Description: "The standard base64 encoding, as defined in RFC 4648.",
				Value:       encodingToLua(L, base64.StdEncoding),
				Type:        encodingTypeName,
			},
			{
				Name:        "raw_std_encoding",
				Description: "The standard raw, unpadded base64 encoding, as defined in RFC 4648.",
				Value:       encodingToLua(L, base64.RawStdEncoding),
				Type:        encodingTypeName,
			},
			{
				Name:        "url_encoding",
				Description: "The alternate base64 encoding defined in RFC 4648. It is typically used in URLs and file names.",
				Value:       encodingToLua(L, base64.URLEncoding),
				Type:        encodingTypeName,
			},
			{
				Name:        "raw_url_encoding",
				Description: "The alternate raw, unpadded base64 encoding defined in RFC 4648. It is typically used in URLs and file names.",
				Value:       encodingToLua(L, base64.RawURLEncoding),
				Type:        encodingTypeName,
			},
		},
		Funcs: []*luadoc.Func{
			{
				Name:        "decode",
				Description: "Decodes a base64 string.",
				Value:       decode,
				Params: []*luadoc.Param{
					{
						Name:        "value",
						Description: "The base64 string to decode.",
						Type:        luadoc.String,
					},
					{
						Name:        "encoding",
						Description: "The encoding to use. Defaults to `std_encoding`.",
						Type:        encodingTypeName,
						Optional:    true,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "decoded",
						Description: "The decoded string.",
						Type:        luadoc.String,
					},
				},
			},
			{
				Name:        "encode",
				Description: "Encodes a string to base64.",
				Value:       encode,
				Params: []*luadoc.Param{
					{
						Name:        "value",
						Description: "The string to encode.",
						Type:        luadoc.String,
					},
					{
						Name:        "encoding",
						Description: "The encoding to use. Defaults to `std_encoding`.",
						Type:        encodingTypeName,
						Optional:    true,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "encoded",
						Description: "The encoded string.",
						Type:        luadoc.String,
					},
				},
			},
		},
	}
}

func encode(L *lua.LState) int {
	value := L.CheckString(1)
	encoding := L.OptUserData(2, encodingToLua(L, base64.StdEncoding)).Value.(*base64.Encoding)
	L.Push(lua.LString(encoding.EncodeToString([]byte(value))))
	return 1
}

func decode(L *lua.LState) int {
	value := L.CheckString(1)
	encoding := L.OptUserData(2, encodingToLua(L, base64.StdEncoding)).Value.(*base64.Encoding)
	decoded, err := encoding.DecodeString(value)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	L.Push(lua.LString(decoded))
	return 1
}
