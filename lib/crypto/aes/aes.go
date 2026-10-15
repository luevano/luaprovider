package aes

import (
	"crypto/aes"
	"fmt"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

func Lib() *luadoc.Lib {
	return &luadoc.Lib{
		Name:        "aes",
		Description: "AES encryption and decryption.",
		Funcs: []*luadoc.Func{
			{
				Name:        "encrypt",
				Description: "Encrypts a single 16-byte block with AES (raw block cipher, no mode or padding). Raises an error unless value is exactly 16 bytes.",
				Value:       encrypt,
				Params: []*luadoc.Param{
					{
						Name:        "key",
						Description: "The key to use for encryption: 16, 24 or 32 bytes (AES-128, AES-192 or AES-256).",
						Type:        luadoc.String,
					},
					{
						Name:        "value",
						Description: "The 16-byte block to encrypt.",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "encrypted",
						Description: "The encrypted 16-byte block.",
						Type:        luadoc.String,
					},
				},
			},
			{
				Name:        "decrypt",
				Description: "Decrypts a single 16-byte block with AES (raw block cipher, no mode or padding). Raises an error unless value is exactly 16 bytes.",
				Value:       decrypt,
				Params: []*luadoc.Param{
					{
						Name:        "key",
						Description: "The key to use for decryption: 16, 24 or 32 bytes (AES-128, AES-192 or AES-256).",
						Type:        luadoc.String,
					},
					{
						Name:        "value",
						Description: "The 16-byte block to decrypt.",
						Type:        luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "decrypted",
						Description: "The decrypted 16-byte block.",
						Type:        luadoc.String,
					},
				},
			},
		},
	}
}

func encrypt(L *lua.LState) int {
	key := L.CheckString(1)
	value := L.CheckString(2)

	cipher, err := aes.NewCipher([]byte(key))
	if err != nil {
		util.Raise(L, fmt.Errorf("aes encrypt: %w", err))
		return 0
	}
	if len(value) != aes.BlockSize {
		util.Raise(L, fmt.Errorf("aes encrypt: value must be exactly %d bytes, got %d", aes.BlockSize, len(value)))
		return 0
	}

	encrypted := make([]byte, len(value))
	cipher.Encrypt(encrypted, []byte(value))

	L.Push(lua.LString(encrypted))
	return 1
}

func decrypt(L *lua.LState) int {
	key := L.CheckString(1)
	value := L.CheckString(2)

	cipher, err := aes.NewCipher([]byte(key))
	if err != nil {
		util.Raise(L, fmt.Errorf("aes decrypt: %w", err))
		return 0
	}
	if len(value) != aes.BlockSize {
		util.Raise(L, fmt.Errorf("aes decrypt: value must be exactly %d bytes, got %d", aes.BlockSize, len(value)))
		return 0
	}

	decrypted := make([]byte, len(value))
	cipher.Decrypt(decrypted, []byte(value))

	L.Push(lua.LString(decrypted))
	return 1
}
