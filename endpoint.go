package gonedrive

import "strings"

// Builds the "$select" query string from a list of wanted results
func EndpointSelect(names ...string) string {
	if len(names) == 0 {
		return ""
	}
	return "$select=" + strings.Join(names, ",")
}

// Helper function, creates URL from path and query.
func EndpointPath(path string, endpoint string, query ...string) string {
	o := "root:/" + path
	if endpoint != "" {
		o += ":/" + endpoint
	}
	if path == "" {
		if endpoint == "" {
			o = "root"
		} else {
			o = "root/" + endpoint
		}
	}
	return o + EndpointQuery(query...)
}

func EndpointQuery(query ...string) (o string) {
	for i, v := range query {
		if i == 0 {
			o += "?"
		} else {
			o += "&"
		}
		o += v
	}
	return
}
