package ui

type CompletionItem struct {
	Text string
	Kind string
}

type JSONContext struct {
	Path    []string
	InKey   bool
	InValue bool
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
	"aggs": {
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
	},
	"aggregations": {
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
	},
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
