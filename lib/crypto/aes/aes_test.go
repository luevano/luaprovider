package aes_test

import (
	"context"
	"testing"

	"github.com/luevano/luaprovider/lib"
	lua "github.com/yuin/gopher-lua"
)

// FIPS-197 appendix C.1 (AES-128)
const (
	key        = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f"
	plaintext  = "\x00\x11\x22\x33\x44\x55\x66\x77\x88\x99\xaa\xbb\xcc\xdd\xee\xff"
	ciphertext = "\x69\xc4\xe0\xd8\x6a\x7b\x04\x30\xd8\xcd\xb7\x80\x70\xb4\xc5\x5a"
)

func TestAES(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	lib.Preload(L, lib.DefaultOptions())
	L.SetContext(context.Background())
	L.SetGlobal("key", lua.LString(key))
	L.SetGlobal("plaintext", lua.LString(plaintext))
	L.SetGlobal("ciphertext", lua.LString(ciphertext))

	script := `
local aes = require("sdk").crypto.aes
assert(aes.encrypt(key, plaintext) == ciphertext, "encrypt block")
assert(aes.decrypt(key, ciphertext) == plaintext, "decrypt block")

local ok, err = pcall(aes.encrypt, key, "short")
assert(not ok and tostring(err):find("exactly 16 bytes"), "short input")
ok, err = pcall(aes.decrypt, key, "short")
assert(not ok and tostring(err):find("exactly 16 bytes"), "short input decrypt")
ok, err = pcall(aes.encrypt, key, plaintext .. plaintext)
assert(not ok and tostring(err):find("got 32"), "longer input")
ok, err = pcall(aes.decrypt, key, ciphertext .. "x")
assert(not ok and tostring(err):find("got 17"), "longer input decrypt")
ok = pcall(aes.encrypt, "badkey", plaintext)
assert(not ok, "bad key size")
`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}
