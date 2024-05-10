package luaprovider

import (
	"encoding/json"
	"strconv"

	"github.com/luevano/libmangal"
	lua "github.com/yuin/gopher-lua"
)

var _ libmangal.Volume = (*luaVolume)(nil)

type luaVolume struct {
	Number float32 `gluamapper:"number"`

	manga *luaManga
	table *lua.LTable
}

func (v *luaVolume) String() string {
	return strconv.FormatFloat(float64(v.Number), 'f', -1, 32)
}

func (v *luaVolume) Manga() libmangal.Manga {
	return v.manga
}

func (v *luaVolume) IntoLValue() lua.LValue {
	return v.table
}

func (v *luaVolume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}

func (v *luaVolume) Info() libmangal.VolumeInfo {
	return libmangal.VolumeInfo{
		Number: v.Number,
	}
}
