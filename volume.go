package luaprovider

import (
	"encoding/json"
	"github.com/luevano/libmangal"
	lua "github.com/yuin/gopher-lua"
	"strconv"
)

var _ libmangal.Volume = (*luaVolume)(nil)

type luaVolume struct {
	Number int `gluamapper:"number"`

	manga *luaManga
	table *lua.LTable
}

func (v luaVolume) String() string {
	return strconv.Itoa(v.Number)
}

func (v luaVolume) Manga() libmangal.Manga {
	return v.manga
}

func (v luaVolume) IntoLValue() lua.LValue {
	return v.table
}

func (v luaVolume) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Info())
}

func (v luaVolume) Info() libmangal.VolumeInfo {
	return libmangal.VolumeInfo{
		Number: v.Number,
	}
}
