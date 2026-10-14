package luaprovider

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	lua "github.com/yuin/gopher-lua"
)

var errCyclic = errors.New("cyclic table")

// cyclic stands for a table found inside itself, it's an error only if a field needs it.
type cyclic struct{}

// mapTable fills the struct dst points to from table, by the fields' lua tags (case-insensitive).
// Missing and nil keys are left alone. Values convert loosely like gluamapper did: numbers and
// bools to strings, numeric strings to numbers, a single value to a one item list.
func mapTable(table *lua.LTable, dst any) (mapstructure.Metadata, error) {
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(cyclicHook, listHook, numberHook),
		WeaklyTypedInput: true,
		Metadata:         &md,
		Result:           dst,
		TagName:          "lua",
	})
	if err != nil {
		return md, err
	}
	err = decoder.Decode(toGoValue(table, map[*lua.LTable]bool{}))
	// mapstructure joins all the field errors under a multiline header
	if err != nil {
		err = fieldErrors(flatten(err, nil))
	}
	return md, err
}

func flatten(err error, errs []error) []error {
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		return append(errs, err)
	}
	for _, err := range joined.Unwrap() {
		errs = flatten(err, errs)
	}
	return errs
}

// fieldErrors are the errors of every field that failed, in one line.
type fieldErrors []error

func (e fieldErrors) Error() string {
	msgs := make([]string, len(e))
	for i, err := range e {
		msgs[i] = err.Error()
	}
	return strings.Join(msgs, "; ")
}

func (e fieldErrors) Unwrap() []error { return e }

// toGoValue converts a lua value for mapstructure: a table with only 1..n keys is a list (nil
// holes stay nil), any other table a map with string keys. path are the tables value is inside of.
func toGoValue(value lua.LValue, path map[*lua.LTable]bool) any {
	switch value := value.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(value)
	case lua.LNumber:
		return float64(value)
	case lua.LString:
		return string(value)
	case *lua.LTable:
		if path[value] {
			return cyclic{}
		}
		path[value] = true
		defer delete(path, value)

		n := value.MaxN()
		isList := true
		value.ForEach(func(k, _ lua.LValue) {
			i, ok := k.(lua.LNumber)
			if !ok || float64(i) != float64(int(i)) || int(i) < 1 || int(i) > n {
				isList = false
			}
		})
		if isList && n > 0 {
			list := make([]any, n)
			for i := range n {
				list[i] = toGoValue(value.RawGetInt(i+1), path)
			}
			return list
		}

		m := make(map[string]any)
		value.ForEach(func(k, v lua.LValue) {
			m[k.String()] = toGoValue(v, path)
		})
		return m
	}
	// functions, userdata...: mapstructure errors if a field gets one
	return value
}

func cyclicHook(_, _ reflect.Type, data any) (any, error) {
	if _, ok := data.(cyclic); ok {
		return nil, errCyclic
	}
	return data, nil
}

// listHook makes a list of the 1..n keys of a mixed table, e.g. {"a", x = "b"} is {"a"}.
func listHook(_, to reflect.Type, data any) (any, error) {
	m, ok := data.(map[string]any)
	if !ok || to.Kind() != reflect.Slice {
		return data, nil
	}
	type item struct {
		i   int
		key string
	}
	var items []item
	for k := range m {
		if i, err := strconv.Atoi(k); err == nil && i > 0 {
			items = append(items, item{i, k})
		}
	}
	if len(items) == 0 {
		return data, nil
	}
	slices.SortFunc(items, func(a, b item) int { return a.i - b.i })
	list := make([]any, len(items))
	for i, it := range items {
		list[i] = m[it.key]
	}
	return list, nil
}

// numberHook trims scraped numeric strings and lets whole number fields take "12.0".
func numberHook(_, to reflect.Type, data any) (any, error) {
	s, ok := data.(string)
	if !ok {
		return data, nil
	}
	switch to.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s = strings.TrimSpace(s)
		if _, err := strconv.ParseInt(s, 0, 64); err != nil {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f, nil
			}
		}
		return s, nil
	case reflect.Float32, reflect.Float64:
		return strings.TrimSpace(s), nil
	}
	return data, nil
}
