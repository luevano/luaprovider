package html

import (
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/PuerkitoBio/goquery"
)

var mdConverter = converter.NewConverter(converter.WithPlugins(
	base.NewBasePlugin(),
	commonmark.NewCommonmarkPlugin(),
	table.NewTablePlugin(),
))

func markdown(selection *goquery.Selection) (string, error) {
	// the converter edits the tree it renders, work on a copy
	clone := selection.Clone()
	parts := make([]string, 0, len(clone.Nodes))
	for _, node := range clone.Nodes {
		out, err := mdConverter.ConvertNode(node)
		if err != nil {
			return "", err
		}
		if s := strings.TrimSpace(string(out)); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "\n\n"), nil
}
