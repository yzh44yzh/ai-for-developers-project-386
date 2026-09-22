// Package home provides the BookMeet root page HTTP handler.
package home

import (
	"net/http"
)

const page = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>BookMeet</title>
</head>
<body>
	<h1>BookMeet</h1>
	<button id="send-data" type="button">Send Data</button>
	<script>
		document.getElementById("send-data").addEventListener("click", () => {
			fetch("/hello", {method: "POST", body: "42"});
		});
	</script>
</body>
</html>
`

// Handler responds with the BookMeet home page.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}
