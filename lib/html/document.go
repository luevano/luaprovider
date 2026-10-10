package html

import (
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/cixtor/readability"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
)

const documentTypeName = libName + "_document"

func parse(L *lua.LState) int {
	value := L.CheckString(1)
	reader := strings.NewReader(value)
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		L.RaiseError("html parse: %s", err)
		return 0
	}

	util.Push(L, document, documentTypeName)
	return 1
}

func documentFind(L *lua.LState) int {
	document := util.Check[*goquery.Document](L, 1)
	selector := L.CheckString(2)

	selection := document.Find(selector)
	util.Push(L, selection, selectionTypeName)
	return 1
}

func documentHtml(L *lua.LState) int {
	document := util.Check[*goquery.Document](L, 1)
	html, err := document.Html()
	if err != nil {
		L.RaiseError("html document: %s", err)
		return 0
	}

	L.Push(lua.LString(html))
	return 1
}

func documentSelection(L *lua.LState) int {
	document := util.Check[*goquery.Document](L, 1)
	selection := document.Selection
	util.Push(L, selection, selectionTypeName)
	return 1
}

func documentMarkdown(L *lua.LState) int {
	document := util.Check[*goquery.Document](L, 1)
	converter := md.NewConverter("", true, nil)

	L.Push(lua.LString(converter.Convert(document.Selection)))
	return 1
}

func documentSimplified(L *lua.LState) int {
	document := util.Check[*goquery.Document](L, 1)
	html, err := document.Html()
	if err != nil {
		L.RaiseError("html document: %s", err)
		return 0
	}

	article, err := readability.New().Parse(strings.NewReader(html), "https://example.com")
	if err != nil {
		L.RaiseError("html simplify: %s", err)
		return 0
	}

	document = goquery.NewDocumentFromNode(article.Node)

	util.Push(L, document, documentTypeName)
	return 1
}
