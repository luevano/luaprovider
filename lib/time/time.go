package time

import (
	"time"

	luadoc "github.com/luevano/gopher-luadoc"
	lua "github.com/yuin/gopher-lua"
)

const (
	timeTypeName = "time"
)

func Lib() *luadoc.Lib {
	return &luadoc.Lib{
		Name:        "time",
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
		},
	}
}

func pushTime(L *lua.LState, t time.Time) {
	ud := L.NewUserData()
	ud.Value = t
	L.SetMetatable(ud, L.GetTypeMetatable(timeTypeName))
	L.Push(ud)
}

func checkTime(L *lua.LState, n int) time.Time {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(time.Time); ok {
		return v
	}
	L.ArgError(n, "time expected")
	return time.Time{}
}

func timeSleep(L *lua.LState) int {
	delay := time.Duration(L.CheckInt64(1))

	select {
	case <-L.Context().Done():
	case <-time.After(delay):
	}

	return 0
}

var timeMethods = map[string]lua.LGFunction{
	"add":        timeAdd,
	"sub":        timeSub,
	"after":      timeAfter,
	"before":     timeBefore,
	"date":       timeDate,
	"day":        timeDay,
	"hour":       timeHour,
	"minute":     timeMinute,
	"month":      timeMonth,
	"nanosecond": timeNanosecond,
	"second":     timeSecond,
	"year":       timeYear,
	"year_day":   timeYearDay,
	"equal":      timeEqual,
	"format":     timeFormat,
	"string":     timeString,
	"clock":      timeClock,
}

func timeAdd(L *lua.LState) int {
	t := checkTime(L, 1)
	d := time.Duration(L.CheckInt64(2))
	pushTime(L, t.Add(d))
	return 1
}

func timeSub(L *lua.LState) int {
	t1 := checkTime(L, 1)
	t2 := checkTime(L, 2)
	L.Push(lua.LNumber(t1.Sub(t2)))
	return 1
}

func timeAfter(L *lua.LState) int {
	t1 := checkTime(L, 1)
	t2 := checkTime(L, 2)
	L.Push(lua.LBool(t1.After(t2)))
	return 1
}

func timeBefore(L *lua.LState) int {
	t1 := checkTime(L, 1)
	t2 := checkTime(L, 2)
	L.Push(lua.LBool(t1.Before(t2)))
	return 1
}

func timeDay(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Day()))
	return 1
}

func timeHour(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Hour()))
	return 1
}

func timeMinute(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Minute()))
	return 1
}

func timeMonth(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Month()))
	return 1
}

func timeNanosecond(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Nanosecond()))
	return 1
}

func timeSecond(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Second()))
	return 1
}

func timeYear(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.Year()))
	return 1
}

func timeYearDay(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LNumber(t.YearDay()))
	return 1
}

func timeEqual(L *lua.LState) int {
	t1 := checkTime(L, 1)
	t2 := checkTime(L, 2)
	L.Push(lua.LBool(t1.Equal(t2)))
	return 1
}

func timeFormat(L *lua.LState) int {
	t := checkTime(L, 1)
	layout := L.CheckString(2)
	L.Push(lua.LString(t.Format(layout)))
	return 1
}

func timeString(L *lua.LState) int {
	t := checkTime(L, 1)
	L.Push(lua.LString(t.String()))
	return 1
}

func timeClock(L *lua.LState) int {
	t := checkTime(L, 1)
	hour, min, sec := t.Clock()
	L.Push(lua.LNumber(hour))
	L.Push(lua.LNumber(min))
	L.Push(lua.LNumber(sec))
	return 3
}

func timeDate(L *lua.LState) int {
	t := checkTime(L, 1)
	year, month, day := t.Date()
	L.Push(lua.LNumber(year))
	L.Push(lua.LNumber(month))
	L.Push(lua.LNumber(day))
	return 3
}

func timeNow(L *lua.LState) int {
	pushTime(L, time.Now())
	return 1
}

func timeParse(L *lua.LState) int {
	layout := L.CheckString(1)
	value := L.CheckString(2)
	t, err := time.Parse(layout, value)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	pushTime(L, t)
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
	pushTime(L, t)
	return 1
}

func timeUnix(L *lua.LState) int {
	sec := L.CheckInt64(1)
	nsec := L.CheckInt64(2)
	t := time.Unix(sec, nsec)
	pushTime(L, t)
	return 1
}
