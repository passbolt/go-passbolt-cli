package commands

import (
	"encoding/json"
	"fmt"
	"strings"
)

// jsonPath returns the value at path in data, or "" if it is missing:
//
//	a.b.c        nested fields
//	[k=v].field  first array element whose field k equals v
func jsonPath(data, path string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return "", fmt.Errorf("invalid JSON: %v", err)
	}
	cur := v
	for _, seg := range tokenize(path) {
		if cur == nil {
			return "", nil
		}
		if strings.HasPrefix(seg, "[") && strings.HasSuffix(seg, "]") {
			arr, ok := cur.([]any)
			if !ok {
				return "", fmt.Errorf("filter %q applied to non-array %T", seg, cur)
			}
			body := seg[1 : len(seg)-1]
			eq := strings.SplitN(body, "=", 2)
			if len(eq) != 2 {
				return "", fmt.Errorf("invalid filter %q (need [key=value])", seg)
			}
			key, want := eq[0], eq[1]
			cur = nil
			for _, el := range arr {
				m, ok := el.(map[string]any)
				if !ok {
					continue
				}
				if stringify(m[key]) == want {
					cur = m
					break
				}
			}
			continue
		}
		m, ok := cur.(map[string]any)
		if !ok {
			return "", fmt.Errorf("field %q applied to non-object %T", seg, cur)
		}
		cur = m[seg]
	}
	return stringify(cur), nil
}

// tokenize splits a path like "a.b[k=v].c" into ["a", "b", "[k=v]", "c"].
func tokenize(path string) []string {
	if path == "" {
		return nil
	}
	var out []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, buf.String())
			buf.Reset()
		}
	}
	i := 0
	for i < len(path) {
		switch c := path[i]; c {
		case '.':
			flush()
			i++
		case '[':
			flush()
			end := strings.IndexByte(path[i:], ']')
			if end < 0 {
				out = append(out, path[i:])
				return out
			}
			out = append(out, path[i:i+end+1])
			i += end + 1
		default:
			buf.WriteByte(c)
			i++
		}
	}
	flush()
	return out
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
