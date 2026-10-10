package time

import (
	"time"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const (
	libName      = "time"
	timeTypeName = libName + "_time"
)

func Lib() *luadoc.Lib {
	return &luadoc.Lib{
		Name:        libName,
		Description: "Time library",
		Vars: []*luadoc.Var{
			{
				Name:        "nanosecond",
				Description: "Duration constant",
				Value:       lua.LNumber(time.Nanosecond),
			},
			{
				Name:        "microsecond",
				Description: "Duration constant. 1000 * nanosecond",
				Value:       lua.LNumber(time.Microsecond),
			},
			{
				Name:        "millisecond",
				Description: "Duration constant. 1000 * microsecond",
				Value:       lua.LNumber(time.Millisecond),
			},
			{
				Name:        "second",
				Description: "Duration constant. 1000 * millisecond",
				Value:       lua.LNumber(time.Second),
			},
			{
				Name:        "minute",
				Description: "Duration constant. 60 * second",
				Value:       lua.LNumber(time.Minute),
			},
			{
				Name:        "hour",
				Description: "Duration constant. 60 * minute",
				Value:       lua.LNumber(time.Hour),
			},
		},
		Funcs: []*luadoc.Func{
			{
				Name:        "sleep",
				Description: "Sleep for the given duration",
				Value:       timeSleep,
				Params: []*luadoc.Param{
					{
						Name: "duration",
						Type: luadoc.Number,
					},
				},
			},
			{
				Name:        "now",
				Description: "Current local time",
				Value:       timeNow,
				Returns: []*luadoc.Param{
					{
						Name: "time",
						Type: timeTypeName,
					},
				},
			},
			{
				Name:        "parse",
				Description: "Parses a formatted string, see https://pkg.go.dev/time#pkg-constants for the layout",
				Value:       timeParse,
				Params: []*luadoc.Param{
					{
						Name: "layout",
						Type: luadoc.String,
					},
					{
						Name: "value",
						Type: luadoc.String,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name: "time",
						Type: timeTypeName,
					},
				},
			},
			{
				Name:        "date",
				Description: "Time of the given date in the local time zone",
				Value:       timeNewDate,
				Params: []*luadoc.Param{
					{
						Name: "year",
						Type: luadoc.Number,
					},
					{
						Name: "month",
						Type: luadoc.Number,
					},
					{
						Name: "day",
						Type: luadoc.Number,
					},
					{
						Name: "hour",
						Type: luadoc.Number,
					},
					{
						Name: "minute",
						Type: luadoc.Number,
					},
					{
						Name: "second",
						Type: luadoc.Number,
					},
					{
						Name: "nanosecond",
						Type: luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name: "time",
						Type: timeTypeName,
					},
				},
			},
			{
				Name:        "unix",
				Description: "Local time of the given unix time",
				Value:       timeUnix,
				Params: []*luadoc.Param{
					{
						Name: "seconds",
						Type: luadoc.Number,
					},
					{
						Name: "nanoseconds",
						Type: luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name: "time",
						Type: timeTypeName,
					},
				},
			},
		},
		Classes: []*luadoc.Class{
			{
				Name:        timeTypeName,
				Description: "A point in time",
				Methods: []*luadoc.Method{
					{
						Name:        "add",
						Description: "Time plus the duration",
						Value:       timeAdd,
						Params: []*luadoc.Param{
							{
								Name: "duration",
								Type: luadoc.Number,
							},
						},
						Returns: []*luadoc.Param{
							{
								Name: "time",
								Type: timeTypeName,
							},
						},
					},
					{
						Name:        "sub",
						Description: "Duration between the times",
						Value:       timeSub,
						Params: []*luadoc.Param{
							{
								Name: "other",
								Type: timeTypeName,
							},
						},
						Returns: []*luadoc.Param{
							{
								Name: "duration",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "after",
						Description: "Whether the time is after the other",
						Value:       timeAfter,
						Params: []*luadoc.Param{
							{
								Name: "other",
								Type: timeTypeName,
							},
						},
						Returns: []*luadoc.Param{
							{
								Name: "after",
								Type: luadoc.Boolean,
							},
						},
					},
					{
						Name:        "before",
						Description: "Whether the time is before the other",
						Value:       timeBefore,
						Params: []*luadoc.Param{
							{
								Name: "other",
								Type: timeTypeName,
							},
						},
						Returns: []*luadoc.Param{
							{
								Name: "before",
								Type: luadoc.Boolean,
							},
						},
					},
					{
						Name:        "equal",
						Description: "Whether both times are the same instant",
						Value:       timeEqual,
						Params: []*luadoc.Param{
							{
								Name: "other",
								Type: timeTypeName,
							},
						},
						Returns: []*luadoc.Param{
							{
								Name: "equal",
								Type: luadoc.Boolean,
							},
						},
					},
					{
						Name:        "date",
						Description: "Year, month and day",
						Value:       timeDate,
						Returns: []*luadoc.Param{
							{
								Name: "year",
								Type: luadoc.Number,
							},
							{
								Name: "month",
								Type: luadoc.Number,
							},
							{
								Name: "day",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "clock",
						Description: "Hour, minute and second",
						Value:       timeClock,
						Returns: []*luadoc.Param{
							{
								Name: "hour",
								Type: luadoc.Number,
							},
							{
								Name: "minute",
								Type: luadoc.Number,
							},
							{
								Name: "second",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "year",
						Description: "Year",
						Value:       timeYear,
						Returns: []*luadoc.Param{
							{
								Name: "year",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "month",
						Description: "Month",
						Value:       timeMonth,
						Returns: []*luadoc.Param{
							{
								Name: "month",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "day",
						Description: "Day",
						Value:       timeDay,
						Returns: []*luadoc.Param{
							{
								Name: "day",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "hour",
						Description: "Hour",
						Value:       timeHour,
						Returns: []*luadoc.Param{
							{
								Name: "hour",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "minute",
						Description: "Minute",
						Value:       timeMinute,
						Returns: []*luadoc.Param{
							{
								Name: "minute",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "second",
						Description: "Second",
						Value:       timeSecond,
						Returns: []*luadoc.Param{
							{
								Name: "second",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "nanosecond",
						Description: "Nanosecond",
						Value:       timeNanosecond,
						Returns: []*luadoc.Param{
							{
								Name: "nanosecond",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "year_day",
						Description: "Day of the year",
						Value:       timeYearDay,
						Returns: []*luadoc.Param{
							{
								Name: "day",
								Type: luadoc.Number,
							},
						},
					},
					{
						Name:        "format",
						Description: "Formats the time, see https://pkg.go.dev/time#pkg-constants for the layout",
						Value:       timeFormat,
						Params: []*luadoc.Param{
							{
								Name: "layout",
								Type: luadoc.String,
							},
						},
						Returns: []*luadoc.Param{
							{
								Name: "formatted",
								Type: luadoc.String,
							},
						},
					},
					{
						Name:        "string",
						Description: "Default string representation",
						Value:       timeString,
						Returns: []*luadoc.Param{
							{
								Name: "string",
								Type: luadoc.String,
							},
						},
					},
				},
			},
		},
	}
}

func timeSleep(L *lua.LState) int {
	delay := time.Duration(L.CheckInt64(1))

	select {
	case <-L.Context().Done():
	case <-time.After(delay):
	}

	return 0
}

func timeAdd(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	d := time.Duration(L.CheckInt64(2))
	util.Push(L, t.Add(d), timeTypeName)
	return 1
}

func timeSub(L *lua.LState) int {
	t1 := util.Check[time.Time](L, 1)
	t2 := util.Check[time.Time](L, 2)
	L.Push(lua.LNumber(t1.Sub(t2)))
	return 1
}

func timeAfter(L *lua.LState) int {
	t1 := util.Check[time.Time](L, 1)
	t2 := util.Check[time.Time](L, 2)
	L.Push(lua.LBool(t1.After(t2)))
	return 1
}

func timeBefore(L *lua.LState) int {
	t1 := util.Check[time.Time](L, 1)
	t2 := util.Check[time.Time](L, 2)
	L.Push(lua.LBool(t1.Before(t2)))
	return 1
}

func timeDay(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Day()))
	return 1
}

func timeHour(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Hour()))
	return 1
}

func timeMinute(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Minute()))
	return 1
}

func timeMonth(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Month()))
	return 1
}

func timeNanosecond(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Nanosecond()))
	return 1
}

func timeSecond(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Second()))
	return 1
}

func timeYear(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.Year()))
	return 1
}

func timeYearDay(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LNumber(t.YearDay()))
	return 1
}

func timeEqual(L *lua.LState) int {
	t1 := util.Check[time.Time](L, 1)
	t2 := util.Check[time.Time](L, 2)
	L.Push(lua.LBool(t1.Equal(t2)))
	return 1
}

func timeFormat(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	layout := L.CheckString(2)
	L.Push(lua.LString(t.Format(layout)))
	return 1
}

func timeString(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	L.Push(lua.LString(t.String()))
	return 1
}

func timeClock(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	hour, min, sec := t.Clock()
	L.Push(lua.LNumber(hour))
	L.Push(lua.LNumber(min))
	L.Push(lua.LNumber(sec))
	return 3
}

func timeDate(L *lua.LState) int {
	t := util.Check[time.Time](L, 1)
	year, month, day := t.Date()
	L.Push(lua.LNumber(year))
	L.Push(lua.LNumber(month))
	L.Push(lua.LNumber(day))
	return 3
}

func timeNow(L *lua.LState) int {
	util.Push(L, time.Now(), timeTypeName)
	return 1
}

func timeParse(L *lua.LState) int {
	layout := L.CheckString(1)
	value := L.CheckString(2)
	t, err := time.Parse(layout, value)
	if err != nil {
		L.RaiseError("time parse %q: %s", value, err)
		return 0
	}
	util.Push(L, t, timeTypeName)
	return 1
}

func timeNewDate(L *lua.LState) int {
	year := L.CheckInt(1)
	month := L.CheckInt(2)
	day := L.CheckInt(3)
	hour := L.CheckInt(4)
	min := L.CheckInt(5)
	sec := L.CheckInt(6)
	nsec := L.CheckInt(7)
	t := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.Local)
	util.Push(L, t, timeTypeName)
	return 1
}

func timeUnix(L *lua.LState) int {
	sec := L.CheckInt64(1)
	nsec := L.CheckInt64(2)
	t := time.Unix(sec, nsec)
	util.Push(L, t, timeTypeName)
	return 1
}
