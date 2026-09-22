// Package hello provides a trivial greeting HTTP handler.
package hello

import (
	"net/http"
)

const page = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Hello</title>
</head>
<body>
	<h1>Hello</h1>
</body>
</html>
`

// Handler responds with an HTML page greeting.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}
