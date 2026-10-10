package html

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/cixtor/readability"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const selectionTypeName = libName + "_selection"

func selectionFind(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)

	selection = selection.Find(selector)
	util.Push(L, selection, selectionTypeName)
	return 1
}

func selectionEach(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	callback := L.CheckFunction(2)

	util.Push(L, selection.Each(func(i int, selection *goquery.Selection) {
		L.Push(callback)
		util.Push(L, selection, selectionTypeName)
		L.Push(lua.LNumber(i))

		if err := L.PCall(2, 0, nil); err != nil {
			L.RaiseError("html each: %s", err)
			return
		}
	}), selectionTypeName)

	return 1
}

func selectionText(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	L.Push(lua.LString(selection.Text()))
	return 1
}

func selectionHtml(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	html, err := selection.Html()
	if err != nil {
		L.RaiseError("html selection: %s", err)
		return 0
	}

	L.Push(lua.LString(html))
	return 1
}

func selectionFirst(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.First(), selectionTypeName)
	return 1
}

func selectionLast(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Last(), selectionTypeName)
	return 1
}

func selectionParent(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Parent(), selectionTypeName)
	return 1
}

func selectionEq(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	index := L.CheckInt(2)
	util.Push(L, selection.Eq(index), selectionTypeName)
	return 1
}

func selectionAttr(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	key := L.CheckString(2)
	attr, exists := selection.Attr(key)
	L.Push(lua.LString(attr))
	L.Push(lua.LBool(exists))

	return 2
}

func selectionAttrOr(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	key := L.CheckString(2)
	defaultValue := L.CheckString(3)
	attr, exists := selection.Attr(key)
	if !exists {
		L.Push(lua.LString(defaultValue))
		return 1
	}

	L.Push(lua.LString(attr))
	return 1
}

func selectionHasClass(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	class := L.CheckString(2)
	L.Push(lua.LBool(selection.HasClass(class)))
	return 1
}

func selectionAddClass(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	class := L.CheckString(2)
	selection.AddClass(class)
	return 0
}

func selectionRemoveClass(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	class := L.CheckString(2)
	selection.RemoveClass(class)
	return 0
}

func selectionToggleClass(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	class := L.CheckString(2)
	selection.ToggleClass(class)
	return 0
}

func selectionNext(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Next(), selectionTypeName)
	return 1
}

func selectionNextAll(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.NextAll(), selectionTypeName)
	return 1
}

func selectionNextUntil(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.NextUntil(selector), selectionTypeName)
	return 1
}

func selectionPrev(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Prev(), selectionTypeName)
	return 1
}

func selectionPrevAll(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.PrevAll(), selectionTypeName)
	return 1
}

func selectionPrevUntil(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.PrevUntil(selector), selectionTypeName)
	return 1
}

func selectionSiblings(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Siblings(), selectionTypeName)
	return 1
}

func selectionChildren(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Children(), selectionTypeName)
	return 1
}

func selectionContents(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Contents(), selectionTypeName)
	return 1
}

func selectionFilter(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.Filter(selector), selectionTypeName)
	return 1
}

func selectionRemove(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.Not(selector), selectionTypeName)
	return 1
}

func selectionIs(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	L.Push(lua.LBool(selection.Is(selector)))
	return 1
}

func selectionFindSelection(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	other := util.Check[*goquery.Selection](L, 2)
	util.Push(L, selection.FindSelection(other), selectionTypeName)
	return 1
}

func selectionMap(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	callback := L.CheckFunction(2)

	table := L.NewTable()

	selection.Each(func(i int, selection *goquery.Selection) {
		L.Push(callback)
		util.Push(L, selection, selectionTypeName)
		if err := L.PCall(1, 1, nil); err != nil {
			L.RaiseError("html map: %s", err)
			return
		}

		value := L.Get(-1)
		L.Pop(1)

		table.RawSetInt(i+1, value)
	})

	L.Push(table)
	return 1
}

func selectionClosest(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.Closest(selector), selectionTypeName)
	return 1
}

func selectionParents(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.Parents(), selectionTypeName)
	return 1
}

func selectionParentsUntil(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.ParentsUntil(selector), selectionTypeName)
	return 1
}

func selectionSlice(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	start := L.CheckInt(2)
	end := L.CheckInt(3)
	util.Push(L, selection.Slice(start, end), selectionTypeName)
	return 1
}

func selectionTerminate(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.End(), selectionTypeName)
	return 1
}

func selectionAdd(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	selector := L.CheckString(2)
	util.Push(L, selection.Add(selector), selectionTypeName)
	return 1
}

func selectionLength(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	L.Push(lua.LNumber(selection.Length()))
	return 1
}

func selectionAddSelection(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	other := util.Check[*goquery.Selection](L, 2)
	util.Push(L, selection.AddSelection(other), selectionTypeName)
	return 1
}

func selectionAddBack(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	util.Push(L, selection.AddBack(), selectionTypeName)
	return 1
}

func selectionMarkdown(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)
	converter := md.NewConverter("", true, nil)

	L.Push(lua.LString(converter.Convert(selection)))
	return 1
}

func selectionSimplified(L *lua.LState) int {
	selection := util.Check[*goquery.Selection](L, 1)

	html, err := selection.Html()
	if err != nil {
		L.RaiseError("html selection: %s", err)
		return 0
	}

	article, err := readability.New().Parse(strings.NewReader(html), "https://example.com")
	if err != nil {
		L.RaiseError("html simplify: %s", err)
		return 0
	}

	document := goquery.NewDocumentFromNode(article.Node)

	util.Push(L, document.Selection, selectionTypeName)
	return 1
}
