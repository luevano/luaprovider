package util

import (
	"slices"

	luadoc "github.com/luevano/gopher-luadoc"
	"github.com/luevano/luaprovider/util"
	"github.com/samber/lo"
	lua "github.com/yuin/gopher-lua"
)

const libName = "util"

func Lib() *luadoc.Lib {
	const (
		T = "T"
		G = "G"
		Q = "Q"
	)

	predicate := func(t string) *luadoc.Param {
		return &luadoc.Param{
			Name:        "predicate",
			Description: "Predicate function",
			Type: luadoc.Func{
				Params: []*luadoc.Param{
					{
						Name:        "value",
						Description: "Value to check",
						Type:        t,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "Result of the predicate",
						Type:        luadoc.Boolean,
					},
				},
			}.AsType(),
		}
	}

	predicateWithIndex := func(t string) *luadoc.Param {
		return &luadoc.Param{
			Name:        "predicate",
			Description: "Predicate function",
			Type: luadoc.Func{
				Params: []*luadoc.Param{
					{
						Name:        "value",
						Description: "Value to check",
						Type:        t,
					},
					{
						Name:        "index",
						Description: "Index of value",
						Type:        luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "Result of the predicate",
						Type:        luadoc.Boolean,
					},
				},
			}.AsType(),
		}
	}

	return &luadoc.Lib{
		Name:        libName,
		Description: "Functional helpers",
		Funcs: []*luadoc.Func{
			{
				Name:        "chunk",
				Description: "Split list into chunks",
				Value:       chunk,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to split",
						Type:        luadoc.List(T),
					},
					{
						Name:        "size",
						Description: "Chunk size",
						Type:        luadoc.Number,
						Optional:    true,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "chunks",
						Description: "List of chunks",
						Type:        luadoc.List(luadoc.List(T)),
					},
				},
			},
			{
				Name:        "drop_right",
				Description: "Drop n elements from the end of list",
				Value:       dropRight,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to drop from",
						Type:        luadoc.List(T),
					},
					{
						Name:        "n",
						Description: "Number of elements to drop",
						Type:        luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "dropped",
						Description: "List of dropped elements",
						Type:        luadoc.List(T),
					},
				},
			},
			{
				Name:        "drop_right_while",
				Description: "Drop elements from the end of list while predicate is true",
				Value:       dropRightWhile,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to drop from",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
			},
			{
				Name:        "filter",
				Description: "Filter list by predicate",
				Value:       filter,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to filter",
						Type:        luadoc.List(T),
					},
					predicateWithIndex(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "filtered",
						Description: "List of filtered elements",
						Type:        luadoc.List(T),
					},
				},
			},
			{
				Name:        "find",
				Description: "Find first element in list that satisfies predicate.",
				Value:       find,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to search",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "element",
						Description: "First element that satisfies predicate",
						Type:        T,
					},
					{
						Name:        "ok",
						Description: "True if element was found",
						Type:        luadoc.Boolean,
					},
				},
			},
			{
				Name:        "find_index",
				Description: "Find index of first element in list that satisfies predicate.",
				Value:       findIndex,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to search",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "index",
						Description: "Index of first element that satisfies predicate",
						Type:        luadoc.Number,
					},
					{
						Name:        "ok",
						Description: "True if element was found",
						Type:        luadoc.Boolean,
					},
				},
			},
			{
				Name:        "find_last_index",
				Description: "Find index of last element in list that satisfies predicate.",
				Value:       findLastIndex,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to search",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "index",
						Description: "Index of last element that satisfies predicate",
						Type:        luadoc.Number,
					},
					{
						Name:        "ok",
						Description: "True if element was found",
						Type:        luadoc.Boolean,
					},
				},
			},
			{
				Name:        "head",
				Description: "Get first element of list",
				Value:       head,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get first element from",
						Type:        luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "element",
						Description: "First element of list",
						Type:        T,
						Optional:    true,
					},
				},
			},
			{
				Name:        "tail",
				Description: "Get all elements of list except first",
				Value:       tail,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get tail from",
						Type:        luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "tail",
						Description: "List of all elements except first",
						Type:        luadoc.List(T),
						Optional:    true,
					},
				},
			},
			{
				Name:        "last",
				Description: "Get last element of list",
				Value:       last,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get last element from",
						Type:        luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "element",
						Description: "Last element of list",
						Type:        T,
						Optional:    true,
					},
				},
			},
			{
				Name:        "init",
				Description: "Get all elements of list except last",
				Value:       initial,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get initial from",
						Type:        luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "initial",
						Description: "List of all elements except last",
						Type:        luadoc.List(T),
						Optional:    true,
					},
				},
			},
			{
				Name:        "take",
				Description: "Get first n elements of list",
				Value:       take,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get elements from",
						Type:        luadoc.List(T),
					},
					{
						Name:        "n",
						Description: "Number of elements to get",
						Type:        luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "elements",
						Description: "First n elements of list",
						Type:        luadoc.List(T),
						Optional:    true,
					},
				},
			},
			{
				Name:        "drop",
				Description: "Get all elements of list except first n",
				Value:       drop,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get elements from",
						Type:        luadoc.List(T),
					},
					{
						Name:        "n",
						Description: "Number of elements to drop",
						Type:        luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "elements",
						Description: "All elements of list except first n",
						Type:        luadoc.List(T),
						Optional:    true,
					},
				},
			},
			{
				Name:        "drop_while",
				Description: "Drop elements from list while predicate is true",
				Value:       dropWhile,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get elements from",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "elements",
						Description: "Elements of list after predicate is false",
						Type:        luadoc.List(T),
						Optional:    true,
					},
				},
			},
			{
				Name:        "take_while",
				Description: "Take elements from list while predicate is true",
				Value:       takeWhile,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get elements from",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "elements",
						Description: "Elements of list before predicate is false",
						Type:        luadoc.List(T),
						Optional:    true,
					},
				},
			},
			{
				Name:        "reverse",
				Description: "Reverse list",
				Value:       reverse,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to reverse",
						Type:        luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "reversed",
						Description: "Reversed list",
						Type:        luadoc.List(T),
					},
				},
			},
			{
				Name:        "slice",
				Description: "Get slice of list",
				Value:       slice,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to get slice from",
						Type:        luadoc.List(T),
					},
					{
						Name:        "start",
						Description: "Start index of slice",
						Type:        luadoc.Number,
					},
					{
						Name:        "finish",
						Description: "End index of slice. Defaults to length of list",
						Type:        luadoc.Number,
						Optional:    true,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "slice",
						Description: "Slice of list",
						Type:        luadoc.List(T),
					},
				},
			},
			{
				Name:        "sort",
				Description: "Returns a sorted list",
				Value:       sort,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to sort",
						Type:        luadoc.List(T),
					},
					{
						Name:        "cmp",
						Description: "Comparison function. Defaults to <",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "a",
									Type: T,
								},
								{
									Name: "b",
									Type: T,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: luadoc.Boolean,
								},
							},
						}.AsType(),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "sorted",
						Description: "Sorted list",
						Type:        luadoc.List(T),
					},
				},
			},
			{
				Name:        "map",
				Description: "Maps a function over a table values",
				Value:       map_,
				Generics:    []string{T, G, Q},
				Params: []*luadoc.Param{
					{
						Name:        "table",
						Description: "Table to map over",
						Type:        luadoc.Map(T, G),
					},
					{
						Name:        "func",
						Description: "Mapping function",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "value",
									Type: G,
								},
								{
									Name: "key",
									Type: T,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: Q,
								},
							},
						}.AsType(),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "mapped",
						Description: "Mapped list",
						Type:        luadoc.Map(T, Q),
					},
				},
			},
			{
				Name:        "map_async",
				Description: "Same as map, the function can't run concurrently in lua",
				Value:       mapAsync,
				Generics:    []string{T, G, Q},
				Params: []*luadoc.Param{
					{
						Name:        "table",
						Description: "Table to map values over",
						Type:        luadoc.Map(T, G),
					},
					{
						Name:        "func",
						Description: "Mapping function. Takes a value of each key",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "value",
									Type: G,
								},
								{
									Name: "key",
									Type: T,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: Q,
								},
							},
						}.AsType(),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "mapped",
						Description: "Mapped table",
						Type:        luadoc.Map(T, Q),
					},
				},
			},
			{
				Name:        "for_each",
				Description: "Calls a function for each element in a list",
				Value:       forEach,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to iterate over",
						Type:        luadoc.List(T),
					},
					{
						Name:        "func",
						Description: "Function to call",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name:        "value",
									Description: "Value of the element",
									Type:        T,
								},
								{
									Name:        "index",
									Description: "Index of the element in the list",
									Type:        luadoc.Number,
								},
							},
						}.AsType(),
					},
				},
			},
			{
				Name:        "contains_by",
				Description: "Checks if a list contains a value using predicate",
				Value:       containsBy,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to check",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "contains",
						Description: "Whether the list contains the value",
						Type:        luadoc.Boolean,
					},
				},
			},
			{
				Name:        "reduce",
				Description: "Reduces a list to a single value",
				Value:       reduce,
				Generics:    []string{T, G},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to reduce",
						Type:        luadoc.List(T),
					},
					{
						Name:        "func",
						Description: "Function to reduce with",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "accumulator",
									Type: G,
								},
								{
									Name: "value",
									Type: T,
								},
								{
									Name: "index",
									Type: luadoc.Number,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: G,
								},
							},
						}.AsType(),
					},
					{
						Name:        "inital",
						Description: "An initial value",
						Type:        G,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "Result of the reduction",
						Type:        G,
					},
				},
			},
			{
				Name:        "reduce_right",
				Description: "Reduces a list to a single value",
				Value:       reduceRight,
				Generics:    []string{T, G},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to reduce",
						Type:        luadoc.List(T),
					},
					{
						Name:        "func",
						Description: "Function to reduce with",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "accumulator",
									Type: G,
								},
								{
									Name: "value",
									Type: T,
								},
								{
									Name: "index",
									Type: luadoc.Number,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: G,
								},
							},
						}.AsType(),
					},
					{
						Name:        "initial",
						Description: "An initial value",
						Type:        G,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "Result of the reduction",
						Type:        G,
					},
				},
			},
			{
				Name:        "reject",
				Description: "Rejects all elements that match a predicate",
				Value:       reject,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to reject from",
						Type:        luadoc.List(T),
					},
					predicateWithIndex(T),
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "List of rejected elements",
						Type:        luadoc.List(T),
					},
				},
			},
			{
				Name:        "id",
				Description: "Returns the first argument",
				Value:       id,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "value",
						Description: "Value to return",
						Type:        T,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "The first argument",
						Type:        T,
					},
				},
			},
			{
				Name:        "min_by",
				Description: "Returns the minimum value of a list, based on a function",
				Value:       minBy,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to find the minimum value of",
						Type:        luadoc.List(T),
					},
					{
						Name:        "func",
						Description: "Function to use to compare values. Returns true if the first argument is less than the second",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "a",
									Type: T,
								},
								{
									Name: "b",
									Type: T,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: luadoc.Boolean,
								},
							},
						}.AsType(),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "The minimum value",
						Type:        T,
					},
				},
			},
			{
				Name:        "max_by",
				Description: "Returns the maximum value of a list, based on a function",
				Value:       maxBy,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to find the maximum value of",
						Type:        luadoc.List(T),
					},
					{
						Name:        "func",
						Description: "Function to use to compare values. Returns true if the first argument is less than the second",
						Type: luadoc.Func{
							Params: []*luadoc.Param{
								{
									Name: "a",
									Type: T,
								},
								{
									Name: "b",
									Type: T,
								},
							},
							Returns: []*luadoc.Param{
								{
									Name: "result",
									Type: luadoc.Boolean,
								},
							},
						}.AsType(),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name:        "result",
						Description: "The maximum value",
						Type:        T,
					},
				},
			},
			{
				Name:        "some_by",
				Description: "Checks if the predicate is true for any element of list",
				Value:       someBy,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to check",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name: "result",
						Type: luadoc.Boolean,
					},
				},
			},
			{
				Name:        "every_by",
				Description: "Checks if the predicate is true for all elements of list",
				Value:       everyBy,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name:        "list",
						Description: "List to check",
						Type:        luadoc.List(T),
					},
					predicate(T),
				},
				Returns: []*luadoc.Param{
					{
						Name: "result",
						Type: luadoc.Boolean,
					},
				},
			},
			{
				Name:        "sample",
				Description: "Random element of list",
				Value:       sample,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name: "list",
						Type: luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name: "element",
						Type: T,
					},
				},
			},
			{
				Name:        "samples",
				Description: "n random unique elements of list",
				Value:       samples,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name: "list",
						Type: luadoc.List(T),
					},
					{
						Name: "n",
						Type: luadoc.Number,
					},
				},
				Returns: []*luadoc.Param{
					{
						Name: "elements",
						Type: luadoc.List(T),
					},
				},
			},
			{
				Name:        "shuffle",
				Description: "Shuffles list",
				Value:       shuffle,
				Generics:    []string{T},
				Params: []*luadoc.Param{
					{
						Name: "list",
						Type: luadoc.List(T),
					},
				},
				Returns: []*luadoc.Param{
					{
						Name: "shuffled",
						Type: luadoc.List(T),
					},
				},
			},
		},
	}
}

func checkList(L *lua.LState, n int) (list []lua.LValue) {
	table := L.CheckTable(n)
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		list = append(list, value)
	})
	return
}

func chunk(L *lua.LState) int {
	list := checkList(L, 1)
	size := L.OptInt(2, 1)

	chunks := util.ToLValueOrRaise(L, lo.Chunk(list, size))
	L.Push(chunks)
	return 1
}

func drop(L *lua.LState) int {
	list := checkList(L, 1)
	n := L.CheckInt(2)

	dropped := util.ToLValueOrRaise(L, lo.Drop(list, n))
	L.Push(dropped)
	return 1
}

func dropRight(L *lua.LState) int {
	list := checkList(L, 1)
	n := L.CheckInt(2)

	dropped := util.ToLValueOrRaise(L, lo.DropRight(list, n))
	L.Push(dropped)
	return 1
}

func dropWhile(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	dropped := util.ToLValueOrRaise(L, lo.DropWhile(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	}))
	L.Push(dropped)
	return 1
}

func dropRightWhile(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	dropped := util.ToLValueOrRaise(L, lo.DropRightWhile(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	}))
	L.Push(dropped)
	return 1
}

func filter(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	filtered := util.ToLValueOrRaise(L, lo.Filter(list, func(v lua.LValue, i int) bool {
		result := call(L, fn, v, lua.LNumber(i))
		return lua.LVAsBool(result)
	}))
	L.Push(filtered)
	return 1
}

func find(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)
	found, ok := lo.Find(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	})

	lvalue := util.ToLValueOrRaise(L, found)
	L.Push(lvalue)
	L.Push(lua.LBool(ok))
	return 2
}

func findIndex(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)
	_, index, ok := lo.FindIndexOf(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	})

	L.Push(lua.LNumber(index))
	L.Push(lua.LBool(ok))
	return 2
}

func findLastIndex(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)
	_, index, ok := lo.FindLastIndexOf(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	})

	L.Push(lua.LNumber(index))
	L.Push(lua.LBool(ok))

	return 2
}

func head(L *lua.LState) int {
	list := checkList(L, 1)

	var head lua.LValue
	if len(list) == 0 {
		head = nil
	} else {
		head = list[0]
	}

	lvalue := util.ToLValueOrRaise(L, head)
	L.Push(lvalue)
	return 1
}

func tail(L *lua.LState) int {
	list := checkList(L, 1)

	var tail []lua.LValue
	if len(list) == 0 {
		tail = nil
	} else {
		tail = list[1:]
	}

	lvalue := util.ToLValueOrRaise(L, tail)
	L.Push(lvalue)
	return 1
}

func initial(L *lua.LState) int {
	list := checkList(L, 1)

	var initial []lua.LValue
	if len(list) == 0 {
		initial = nil
	} else {
		initial = list[:len(list)-1]
	}

	lvalue := util.ToLValueOrRaise(L, initial)
	L.Push(lvalue)
	return 1
}

func take(L *lua.LState) int {
	list := checkList(L, 1)
	n := L.CheckInt(2)

	var taken []lua.LValue
	if len(list) == 0 {
		taken = nil
	} else {
		taken = list[:n]
	}

	t := util.ToLValueOrRaise(L, taken)
	L.Push(t)
	return 1
}

func takeWhile(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	for i, v := range list {
		result := call(L, fn, v)
		if !lua.LVAsBool(result) {
			list = list[:i]
			break
		}
	}

	l := util.ToLValueOrRaise(L, list)
	L.Push(l)
	return 1
}

func last(L *lua.LState) int {
	list := checkList(L, 1)

	var last lua.LValue
	if len(list) == 0 {
		last = nil
	} else {
		last = list[len(list)-1]
	}

	lvalue := util.ToLValueOrRaise(L, last)
	L.Push(lvalue)
	return 1
}

func reverse(L *lua.LState) int {
	list := checkList(L, 1)

	reversed := util.ToLValueOrRaise(L, lo.Reverse(list))
	L.Push(reversed)
	return 1
}

func slice(L *lua.LState) int {
	list := checkList(L, 1)
	start := L.CheckInt(2)
	end := L.OptInt(3, len(list))

	sliced := util.ToLValueOrRaise(L, lo.Slice(list, start, end))
	L.Push(sliced)
	return 1
}

func sort(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.OptFunction(2, nil)

	lessThan := func(a, b lua.LValue) bool {
		if fn == nil {
			return L.LessThan(a, b)
		}
		return lua.LVAsBool(call(L, fn, a, b))
	}
	slices.SortStableFunc(list, func(a, b lua.LValue) int {
		switch {
		case lessThan(a, b):
			return -1
		case lessThan(b, a):
			return 1
		default:
			return 0
		}
	})

	sorted := util.ToLValueOrRaise(L, list)
	L.Push(sorted)
	return 1
}

func map_(L *lua.LState) int {
	table := L.CheckTable(1)
	fn := L.CheckFunction(2)

	mapped := L.NewTable()

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		result := call(L, fn, value, key)
		mapped.RawSet(key, result)
	})

	L.Push(mapped)
	return 1
}

// mapAsync: Same as map, a lua state can't be used from several goroutines.
func mapAsync(L *lua.LState) int {
	return map_(L)
}

func forEach(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	lo.ForEach(list, func(v lua.LValue, i int) {
		call(L, fn, v, lua.LNumber(i))
	})

	return 0
}

func containsBy(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	contains := lo.ContainsBy(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	})

	L.Push(lua.LBool(contains))
	return 1
}

func reduce(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)
	initial := L.CheckAny(3)

	r := lo.Reduce(list, func(acc lua.LValue, v lua.LValue, i int) lua.LValue {
		result := call(L, fn, acc, v, lua.LNumber(i))
		return result
	}, initial)

	L.Push(r)
	return 1
}

func reduceRight(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)
	initial := L.CheckAny(3)

	r := lo.ReduceRight(list, func(acc lua.LValue, v lua.LValue, i int) lua.LValue {
		result := call(L, fn, acc, v, lua.LNumber(i))
		return result
	}, initial)

	L.Push(r)
	return 1
}

func reject(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	rejected := util.ToLValueOrRaise(L, lo.Reject(list, func(v lua.LValue, i int) bool {
		result := call(L, fn, v, lua.LNumber(i))
		return lua.LVAsBool(result)
	}))
	L.Push(rejected)
	return 1
}

func sample(L *lua.LState) int {
	list := checkList(L, 1)

	sampled := util.ToLValueOrRaise(L, lo.Sample(list))
	L.Push(sampled)
	return 1
}

func samples(L *lua.LState) int {
	list := checkList(L, 1)
	n := L.CheckInt(2)

	sampled := util.ToLValueOrRaise(L, lo.Samples(list, n))
	L.Push(sampled)
	return 1
}

func shuffle(L *lua.LState) int {
	list := checkList(L, 1)

	shuffled := util.ToLValueOrRaise(L, lo.Shuffle(list))
	L.Push(shuffled)
	return 1
}

func someBy(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	L.Push(lua.LBool(lo.SomeBy(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	})))
	return 1
}

func everyBy(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	L.Push(lua.LBool(lo.EveryBy(list, func(v lua.LValue) bool {
		result := call(L, fn, v)
		return lua.LVAsBool(result)
	})))
	return 1
}

func id(L *lua.LState) int {
	value := L.CheckAny(1)
	L.Push(value)
	return 1
}

func minBy(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	min := util.ToLValueOrRaise(L, lo.MinBy(list, func(a, b lua.LValue) bool {
		result := call(L, fn, a, b)
		return lua.LVAsBool(result)
	}))

	L.Push(min)
	return 1
}

func maxBy(L *lua.LState) int {
	list := checkList(L, 1)
	fn := L.CheckFunction(2)

	max := util.ToLValueOrRaise(L, lo.MaxBy(list, func(a, b lua.LValue) bool {
		result := call(L, fn, a, b)
		return lua.LVAsBool(result)
	}))

	L.Push(max)
	return 1
}

// call: Calls fn with args and returns its first result, leaving the stack as it was.
func call(L *lua.LState, fn *lua.LFunction, args ...lua.LValue) lua.LValue {
	L.Push(fn)
	for _, arg := range args {
		L.Push(arg)
	}
	L.Call(len(args), 1)
	result := L.Get(-1)
	L.Pop(1)
	return result
}
