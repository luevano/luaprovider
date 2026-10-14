package html

import (
	"fmt"
	"net/url"
	"strings"

	"codeberg.org/readeck/go-readability/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/luevano/luaprovider/util"
	lua "github.com/yuin/gopher-lua"
	xhtml "golang.org/x/net/html"
)

const documentTypeName = libName + "_document"

func parse(L *lua.LState) int {
	value := L.CheckString(1)
	reader := strings.NewReader(value)
	document, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		util.Raise(L, fmt.Errorf("html parse: %w", err))
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
		util.Raise(L, fmt.Errorf("html document: %w", err))
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
	out, err := markdown(document.Selection)
	if err != nil {
		util.Raise(L, fmt.Errorf("html markdown: %w", err))
		return 0
	}

	L.Push(lua.LString(out))
	return 1
}

func documentSimplified(L *lua.LState) int {
	document := util.Check[*goquery.Document](L, 1)
	html, err := document.Html()
	if err != nil {
		util.Raise(L, fmt.Errorf("html document: %w", err))
		return 0
	}

	simplified, err := simplify(html, document.Url)
	if err != nil {
		util.Raise(L, fmt.Errorf("html simplify: %w", err))
		return 0
	}

	util.Push(L, simplified, documentTypeName)
	return 1
}

// simplify keeps the readable part of the html. Relative links are made
// absolute only when the page url is known.
func simplify(html string, pageURL *url.URL) (*goquery.Document, error) {
	article, err := readability.FromReader(strings.NewReader(html), pageURL)
	if err != nil {
		return nil, err
	}

	node := article.Node
	if node == nil {
		node = &xhtml.Node{Type: xhtml.DocumentNode}
	}
	document := goquery.NewDocumentFromNode(node)
	document.Url = pageURL
	return document, nil
}
