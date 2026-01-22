package ui

var dslSchema = map[string][]CompletionItem{
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
		{Text: "nested", Kind: "keyword"},
		{Text: "ids", Kind: "keyword"},
	},
	"query.bool": {
		{Text: "must", Kind: "keyword"},
		{Text: "should", Kind: "keyword"},
		{Text: "must_not", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "minimum_should_match", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"query.match.*": {
		{Text: "query", Kind: "keyword"},
		{Text: "operator", Kind: "keyword"},
		{Text: "fuzziness", Kind: "keyword"},
		{Text: "analyzer", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"query.range.*": {
		{Text: "gte", Kind: "keyword"},
		{Text: "gt", Kind: "keyword"},
		{Text: "lte", Kind: "keyword"},
		{Text: "lt", Kind: "keyword"},
		{Text: "format", Kind: "keyword"},
		{Text: "boost", Kind: "keyword"},
	},
	"aggs.*": {
		{Text: "terms", Kind: "keyword"},
		{Text: "avg", Kind: "keyword"},
		{Text: "sum", Kind: "keyword"},
		{Text: "min", Kind: "keyword"},
		{Text: "max", Kind: "keyword"},
		{Text: "cardinality", Kind: "keyword"},
		{Text: "date_histogram", Kind: "keyword"},
		{Text: "histogram", Kind: "keyword"},
		{Text: "filter", Kind: "keyword"},
		{Text: "aggs", Kind: "keyword"},
	},
	"aggs.*.terms": {
		{Text: "field", Kind: "keyword"},
		{Text: "size", Kind: "keyword"},
		{Text: "order", Kind: "keyword"},
		{Text: "missing", Kind: "keyword"},
	},
	"sort.*": {
		{Text: "order", Kind: "keyword"},
		{Text: "mode", Kind: "keyword"},
		{Text: "unmapped_type", Kind: "keyword"},
	},
	"highlight": {
		{Text: "fields", Kind: "keyword"},
		{Text: "pre_tags", Kind: "keyword"},
		{Text: "post_tags", Kind: "keyword"},
	},
}

func GetCompletionsForPath(path []string) []CompletionItem {
	if len(path) == 0 {
		return dslSchema[""]
	}

	key := joinPath(path)
	if items, ok := dslSchema[key]; ok {
		return items
	}

	wildcardKey := joinPathWithWildcard(path)
	if items, ok := dslSchema[wildcardKey]; ok {
		return items
	}

	if len(path) > 1 {
		return GetCompletionsForPath(path[:len(path)-1])
	}

	return dslSchema[""]
}

func joinPath(path []string) string {
	result := ""
	for i, p := range path {
		if i > 0 {
			result += "."
		}
		result += p
	}
	return result
}

func joinPathWithWildcard(path []string) string {
	if len(path) < 2 {
		return ""
	}
	result := path[0]
	for i := 1; i < len(path); i++ {
		result += ".*"
	}
	return result
}
