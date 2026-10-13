package luaprovider

import (
	"fmt"
	"reflect"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// mapTable fills the struct dst points to from table, by the fields' lua tags. Missing and nil
// keys are left alone. Values convert loosely (like gluamapper did): numbers and bools to
// strings, numeric strings to numbers, a string to a one item list.
func mapTable(table *lua.LTable, dst any) error {
	v := reflect.ValueOf(dst).Elem()
	for i := range v.NumField() {
		key := v.Type().Field(i).Tag.Get("lua")
		if key == "" {
			continue
		}
		value := table.RawGetString(key)
		if value == lua.LNil {
			continue
		}
		if err := setField(v.Field(i), value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	return nil
}

func setField(field reflect.Value, value lua.LValue) error {
	switch field.Kind() {
	case reflect.String:
		s, err := luaToString(value)
		if err != nil {
			return err
		}
		field.SetString(s)
	case reflect.Float32, reflect.Float64:
		n, err := luaToNumber(value)
		if err != nil {
			return err
		}
		field.SetFloat(n)
	case reflect.Int:
		n, err := luaToNumber(value)
		if err != nil {
			return err
		}
		field.SetInt(int64(n))
	case reflect.Slice: // []string
		list, err := luaToStrings(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(list))
	case reflect.Map: // map[string]string
		table, ok := value.(*lua.LTable)
		if !ok {
			return fmt.Errorf("expected a table, got %s", value.Type())
		}
		m := make(map[string]string)
		var err error
		table.ForEach(func(k, v lua.LValue) {
			if err != nil {
				return
			}
			var key, val string
			if key, err = luaToString(k); err != nil {
				return
			}
			if val, err = luaToString(v); err != nil {
				err = fmt.Errorf("%s: %w", key, err)
				return
			}
			m[key] = val
		})
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(m))
	case reflect.Pointer: // *struct
		table, ok := value.(*lua.LTable)
		if !ok {
			return fmt.Errorf("expected a table, got %s", value.Type())
		}
		ptr := reflect.New(field.Type().Elem())
		if err := mapTable(table, ptr.Interface()); err != nil {
			return err
		}
		field.Set(ptr)
	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}
	return nil
}

func luaToString(value lua.LValue) (string, error) {
	switch value := value.(type) {
	case lua.LString:
		return string(value), nil
	case lua.LNumber:
		return strconv.FormatFloat(float64(value), 'f', -1, 64), nil
	case lua.LBool:
		if value {
			return "1", nil
		}
		return "0", nil
	}
	return "", fmt.Errorf("expected a string, got %s", value.Type())
}

func luaToNumber(value lua.LValue) (float64, error) {
	switch value := value.(type) {
	case lua.LNumber:
		return float64(value), nil
	case lua.LString:
		if value == "" {
			return 0, nil
		}
		return strconv.ParseFloat(string(value), 64)
	case lua.LBool:
		if value {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("expected a number, got %s", value.Type())
}

func luaToStrings(value lua.LValue) ([]string, error) {
	table, ok := value.(*lua.LTable)
	if !ok {
		s, err := luaToString(value)
		if err != nil {
			return nil, err
		}
		return []string{s}, nil
	}
	n := table.MaxN()
	if n == 0 {
		var hasKeys bool
		table.ForEach(func(lua.LValue, lua.LValue) { hasKeys = true })
		if hasKeys {
			return nil, fmt.Errorf("expected a list, got a table with keys")
		}
	}
	list := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		s, err := luaToString(table.RawGetInt(i))
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		list = append(list, s)
	}
	return list, nil
}
