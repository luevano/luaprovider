package luaprovider

import (
	"regexp"

	"github.com/luevano/libmangal/mangadata"
)

var fileExtensionRegex = regexp.MustCompile(`^\.[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]$`)

var _ mangadata.Page = (*luaPage)(nil)

type luaPage struct {
	Ext string `json:"extension" gluamapper:"extension"`

	// URL is the url of the page image
	URL string `json:"url" gluamapper:"url"`

	Headers map[string]string `json:"headers" gluamapper:"headers"`
	Cookies map[string]string `json:"cookies" gluamapper:"cookies"`

	chapter *luaChapter
}

func (p *luaPage) String() string {
	return p.URL
}

func (p *luaPage) Chapter() mangadata.Chapter {
	return p.chapter
}

func (p *luaPage) Extension() string {
	return p.Ext
}
