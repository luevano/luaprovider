package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const libName = "json"

func Lib() *luadoc.Lib {
	return &luadoc.Lib{
		Name:        libName,
		Description: "Provides functions for encoding and decoding JSON.",
		Funcs: []*luadoc.Func{
			{
				Name:        "decode",
				Description: "Decodes a JSON string into a Lua value.",
				Value:       apiDecode,
				Params: []*luadoc.Param{
					{
						Name:        "json",
						Type:        luadoc.String,
						Description: "The JSON string to decode.",
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "value",
						Type:        luadoc.Any,
						Description: "The decoded value.",
					},
				},
			},
			{
				Name:        "encode",
				Description: "Encodes a Lua value into a JSON string.",
				Value:       apiEncode,
				Params: []*luadoc.Param{
					{
						Name:        "value",
						Type:        luadoc.Any,
						Description: "The value to encode.",
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "json",
						Type:        luadoc.String,
						Description: "The encoded JSON string.",
					},
				},
			},
		},
	}
}

func apiDecode(L *lua.LState) int {
	str := L.CheckString(1)

	value, err := decode(L, []byte(str))
	if err != nil {
		util.Raise(L, fmt.Errorf("json decode: %w", err))
		return 0
	}
	L.Push(value)
	return 1
}

func apiEncode(L *lua.LState) int {
	value := L.CheckAny(1)

	data, err := encode(value)
	if err != nil {
		util.Raise(L, fmt.Errorf("json encode: %w", err))
		return 0
	}
	L.Push(lua.LString(string(data)))
	return 1
}

var (
	errNested      = errors.New("cannot encode recursively nested tables to JSON")
	errSparseArray = errors.New("cannot encode sparse array")
	errInvalidKeys = errors.New("cannot encode mixed or invalid key types")
)

type invalidTypeError lua.LValueType

func (i invalidTypeError) Error() string {
	return `cannot encode ` + lua.LValueType(i).String() + ` to JSON`
}

// encode returns the JSON encoding of value.
func encode(value lua.LValue) ([]byte, error) {
	return json.Marshal(jsonValue{
		LValue:  value,
		visited: make(map[*lua.LTable]bool),
	})
}

type jsonValue struct {
	lua.LValue
	visited map[*lua.LTable]bool
}

func (j jsonValue) MarshalJSON() (data []byte, err error) {
	switch converted := j.LValue.(type) {
	case lua.LBool:
		data, err = json.Marshal(bool(converted))
	case lua.LNumber:
		data, err = json.Marshal(float64(converted))
	case *lua.LNilType:
		data = []byte(`null`)
	case lua.LString:
		data, err = json.Marshal(string(converted))
	case *lua.LTable:
		if j.visited[converted] {
			return nil, errNested
		}
		// only the tables being encoded, a table can appear twice if it's not inside itself
		j.visited[converted] = true
		defer delete(j.visited, converted)

		key, value := converted.Next(lua.LNil)

		switch key.Type() {
		case lua.LTNil: // empty table
			data = []byte(`[]`)
		case lua.LTNumber:
			arr := make([]jsonValue, 0, converted.Len())
			expectedKey := lua.LNumber(1)
			for key != lua.LNil {
				if key.Type() != lua.LTNumber {
					err = errInvalidKeys
					return
				}
				if expectedKey != key {
					err = errSparseArray
					return
				}
				arr = append(arr, jsonValue{value, j.visited})
				expectedKey++
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(arr)
		case lua.LTString:
			obj := make(map[string]jsonValue)
			for key != lua.LNil {
				if key.Type() != lua.LTString {
					err = errInvalidKeys
					return
				}
				obj[key.String()] = jsonValue{value, j.visited}
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(obj)
		default:
			err = errInvalidKeys
		}
	default:
		err = invalidTypeError(j.LValue.Type())
	}
	return
}

// decode converts the JSON encoded data to Lua values.
func decode(L *lua.LState, data []byte) (lua.LValue, error) {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return decodeValue(L, value), nil
}

// decodeValue converts the value to a Lua value.
//
// This function only converts values that the encoding/json package decodes to.
// All other values will return lua.LNil.
func decodeValue(L *lua.LState, value any) lua.LValue {
	switch converted := value.(type) {
	case bool:
		return lua.LBool(converted)
	case float64:
		return lua.LNumber(converted)
	case string:
		return lua.LString(converted)
	case json.Number:
		return lua.LString(converted)
	case []any:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			arr.Append(decodeValue(L, item))
		}
		return arr
	case map[string]any:
		tbl := L.CreateTable(0, len(converted))
		// tables iterate in insertion order, sorted keys make it the same every time
		for _, key := range slices.Sorted(maps.Keys(converted)) {
			tbl.RawSetH(lua.LString(key), decodeValue(L, converted[key]))
		}
		return tbl
	case nil:
		return lua.LNil
	}

	return lua.LNil
}
