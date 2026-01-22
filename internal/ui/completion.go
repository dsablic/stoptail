package ui

import "strings"

type CompletionItem struct {
	Text string
	Kind string
}

type JSONContext struct {
	Path    []string
	InKey   bool
	InValue bool
}

var aggKeywords = []CompletionItem{
	{Text: "terms", Kind: "keyword"},
	{Text: "avg", Kind: "keyword"},
	{Text: "sum", Kind: "keyword"},
	{Text: "min", Kind: "keyword"},
	{Text: "max", Kind: "keyword"},
	{Text: "cardinality", Kind: "keyword"},
	{Text: "value_count", Kind: "keyword"},
	{Text: "stats", Kind: "keyword"},
	{Text: "date_histogram", Kind: "keyword"},
	{Text: "histogram", Kind: "keyword"},
	{Text: "range", Kind: "keyword"},
	{Text: "filter", Kind: "keyword"},
	{Text: "nested", Kind: "keyword"},
}

var dslKeywords = map[string][]CompletionItem{
	"": {
		{Text: "query", Kind: "keyword"},
		{Text: "aggs", Kind: "keyword"},
		{Text: "aggregations", Kind: "keyword"},
		{Text: "size", Kind: "keyword"},
		{Text: "from", Kind: "keyword"},
		{Text: "sort", Kind: "keyword"},
		{Text: "_source", Kind: "keyword"},
		{Text: "highlight", Kind: "keyword"},
		{Text: "track_total_hits", Kind: "keyword"},
	},
	"query": {
		{Text: "bool", Kind: "keyword"},
		{Text: "match", Kind: "keyword"},
		{Text: "match_all", Kind: "keyword"},
		{Text: "match_phrase", Kind: "keyword"},
		{Text: "multi_match", Kind: "keyword"},
		{Text: "term", Kind: "keyword"},
		{Text: "terms", Kind: "keyword"},
		{Text: "range", Kind: "keyword"},
		{Text: "exists", Kind: "keyword"},
		{Text: "prefix", Kind: "keyword"},
		{Text: "wildcard", Kind: "keyword"},
		{Text: "regexp", Kind: "keyword"},
		{Text: "fuzzy", Kind: "keyword"},
		{Text: "nested", Kind: "keyword"},
		{Text: "ids", Kind: "keyword"},
	},
	"bool": {
		{Text: "must", Kind: "keyword"},
		{Text: "should", Kind: "keyword"},
		{Text: "must_not", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "minimum_should_match", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"aggs":         aggKeywords,
	"aggregations": aggKeywords,
	"match": {
		{Text: "query", Kind: "keyword"},
		{Text: "operator", Kind: "keyword"},
		{Text: "fuzziness", Kind: "keyword"},
		{Text: "analyzer", Kind: "keyword"},
	},
	"range": {
		{Text: "gte", Kind: "keyword"},
		{Text: "gt", Kind: "keyword"},
		{Text: "lte", Kind: "keyword"},
		{Text: "lt", Kind: "keyword"},
		{Text: "format", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"sort": {
		{Text: "order", Kind: "keyword"},
		{Text: "mode", Kind: "keyword"},
		{Text: "nested", Kind: "keyword"},
		{Text: "unmapped_type", Kind: "keyword"},
	},
	"highlight": {
		{Text: "fields", Kind: "keyword"},
		{Text: "pre_tags", Kind: "keyword"},
		{Text: "post_tags", Kind: "keyword"},
		{Text: "number_of_fragments", Kind: "keyword"},
	},
}

func ParseJSONContext(text string) JSONContext {
	ctx := JSONContext{}
	var path []string
	var currentKey string
	var inString bool
	var afterColon bool
	var depth int

	for i := 0; i < len(text); i++ {
		c := text[i]

		if inString {
			if c == '"' && (i == 0 || text[i-1] != '\\') {
				inString = false
				if afterColon {
					afterColon = false
				}
			} else {
				currentKey += string(c)
			}
			continue
		}

		switch c {
		case '"':
			inString = true
			currentKey = ""
		case ':':
			if currentKey != "" {
				path = append(path, currentKey)
			}
			afterColon = true
			currentKey = ""
		case '{':
			depth++
			afterColon = false
		case '}':
			depth--
			if len(path) > 0 {
				path = path[:len(path)-1]
			}
			afterColon = false
		case '[':
			afterColon = false
		case ']':
			afterColon = false
		case ',':
			afterColon = false
			currentKey = ""
		}
	}

	ctx.Path = path
	ctx.InKey = !afterColon && (inString || depth > 0)
	ctx.InValue = afterColon

	return ctx
}

func GetKeywordsForContext(path []string) []CompletionItem {
	if len(path) == 0 {
		return dslKeywords[""]
	}
	lastKey := path[len(path)-1]
	if items, ok := dslKeywords[lastKey]; ok {
		return items
	}
	for i := len(path) - 2; i >= 0; i-- {
		if items, ok := dslKeywords[path[i]]; ok {
			return items
		}
	}
	return dslKeywords[""]
}

type CompletionState struct {
	Active      bool
	Items       []CompletionItem
	Filtered    []CompletionItem
	SelectedIdx int
	TriggerCol  int
	Query       string
}

func (c *CompletionState) Filter(query string) {
	c.Query = query
	c.Filtered = nil
	c.SelectedIdx = 0
	query = strings.ToLower(query)
	for _, item := range c.Items {
		if strings.HasPrefix(strings.ToLower(item.Text), query) {
			c.Filtered = append(c.Filtered, item)
		}
	}
	if len(c.Filtered) == 0 {
		c.Active = false
	}
}

func (c *CompletionState) MoveUp() {
	if c.SelectedIdx > 0 {
		c.SelectedIdx--
	}
}

func (c *CompletionState) MoveDown() {
	if c.SelectedIdx < len(c.Filtered)-1 {
		c.SelectedIdx++
	}
}

func (c *CompletionState) Selected() *CompletionItem {
	if c.SelectedIdx >= 0 && c.SelectedIdx < len(c.Filtered) {
		return &c.Filtered[c.SelectedIdx]
	}
	return nil
}

func (c *CompletionState) Close() {
	c.Active = false
	c.Items = nil
	c.Filtered = nil
	c.SelectedIdx = 0
	c.Query = ""
}
