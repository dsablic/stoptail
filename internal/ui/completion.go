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

var queryKeywords = []CompletionItem{
	{Text: "bool", Kind: "query"},
	{Text: "match", Kind: "query"},
	{Text: "match_all", Kind: "query"},
	{Text: "match_none", Kind: "query"},
	{Text: "match_phrase", Kind: "query"},
	{Text: "match_phrase_prefix", Kind: "query"},
	{Text: "multi_match", Kind: "query"},
	{Text: "term", Kind: "query"},
	{Text: "terms", Kind: "query"},
	{Text: "range", Kind: "query"},
	{Text: "exists", Kind: "query"},
	{Text: "prefix", Kind: "query"},
	{Text: "wildcard", Kind: "query"},
	{Text: "regexp", Kind: "query"},
	{Text: "fuzzy", Kind: "query"},
	{Text: "ids", Kind: "query"},
	{Text: "nested", Kind: "query"},
	{Text: "has_child", Kind: "query"},
	{Text: "has_parent", Kind: "query"},
	{Text: "parent_id", Kind: "query"},
	{Text: "geo_bounding_box", Kind: "query"},
	{Text: "geo_distance", Kind: "query"},
	{Text: "geo_shape", Kind: "query"},
	{Text: "query_string", Kind: "query"},
	{Text: "simple_query_string", Kind: "query"},
	{Text: "script", Kind: "query"},
	{Text: "percolate", Kind: "query"},
	{Text: "wrapper", Kind: "query"},
	{Text: "span_term", Kind: "query"},
	{Text: "span_multi", Kind: "query"},
	{Text: "span_first", Kind: "query"},
	{Text: "span_near", Kind: "query"},
	{Text: "span_or", Kind: "query"},
	{Text: "span_not", Kind: "query"},
	{Text: "span_containing", Kind: "query"},
	{Text: "span_within", Kind: "query"},
	{Text: "constant_score", Kind: "query"},
	{Text: "dis_max", Kind: "query"},
	{Text: "function_score", Kind: "query"},
	{Text: "boosting", Kind: "query"},
}

var aggKeywords = []CompletionItem{
	{Text: "terms", Kind: "agg"},
	{Text: "avg", Kind: "agg"},
	{Text: "sum", Kind: "agg"},
	{Text: "min", Kind: "agg"},
	{Text: "max", Kind: "agg"},
	{Text: "cardinality", Kind: "agg"},
	{Text: "value_count", Kind: "agg"},
	{Text: "stats", Kind: "agg"},
	{Text: "extended_stats", Kind: "agg"},
	{Text: "percentiles", Kind: "agg"},
	{Text: "percentile_ranks", Kind: "agg"},
	{Text: "date_histogram", Kind: "agg"},
	{Text: "histogram", Kind: "agg"},
	{Text: "range", Kind: "agg"},
	{Text: "date_range", Kind: "agg"},
	{Text: "filter", Kind: "agg"},
	{Text: "filters", Kind: "agg"},
	{Text: "nested", Kind: "agg"},
	{Text: "reverse_nested", Kind: "agg"},
	{Text: "global", Kind: "agg"},
	{Text: "missing", Kind: "agg"},
	{Text: "sampler", Kind: "agg"},
	{Text: "significant_terms", Kind: "agg"},
	{Text: "top_hits", Kind: "agg"},
	{Text: "geo_bounds", Kind: "agg"},
	{Text: "geo_centroid", Kind: "agg"},
	{Text: "composite", Kind: "agg"},
	{Text: "aggs", Kind: "agg"},
	{Text: "aggregations", Kind: "agg"},
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
		{Text: "timeout", Kind: "keyword"},
		{Text: "terminate_after", Kind: "keyword"},
		{Text: "min_score", Kind: "keyword"},
		{Text: "explain", Kind: "keyword"},
		{Text: "version", Kind: "keyword"},
		{Text: "seq_no_primary_term", Kind: "keyword"},
		{Text: "stored_fields", Kind: "keyword"},
		{Text: "docvalue_fields", Kind: "keyword"},
		{Text: "script_fields", Kind: "keyword"},
		{Text: "collapse", Kind: "keyword"},
		{Text: "search_after", Kind: "keyword"},
		{Text: "pit", Kind: "keyword"},
		{Text: "runtime_mappings", Kind: "keyword"},
		{Text: "post_filter", Kind: "keyword"},
		{Text: "rescore", Kind: "keyword"},
		{Text: "suggest", Kind: "keyword"},
		{Text: "profile", Kind: "keyword"},
		{Text: "indices_boost", Kind: "keyword"},
	},
	"query": queryKeywords,
	"bool": {
		{Text: "must", Kind: "keyword"},
		{Text: "should", Kind: "keyword"},
		{Text: "must_not", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "minimum_should_match", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"must":     queryKeywords,
	"should":   queryKeywords,
	"must_not": queryKeywords,
	"filter":   queryKeywords,
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
