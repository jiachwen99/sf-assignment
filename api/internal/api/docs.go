package api

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec []byte

// Swagger UI comes from a CDN, and the version is pinned exactly so both assets
// can carry integrity hashes. A floating major version cannot have them, and
// without them a compromised CDN runs whatever it likes on this origin.
const swaggerPage = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>TODO API</title>
    <link rel="stylesheet"
          href="https://unpkg.com/swagger-ui-dist@5.30.1/swagger-ui.css"
          integrity="sha384-++DMKo1369T5pxDNqojF1F91bYxYiT1N7b1M15a7oCzEodfljztKlApQoH6eQSKI"
          crossorigin="anonymous">
  </head>
  <body>
    <div id="ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.30.1/swagger-ui-bundle.js"
            integrity="sha384-D12ttDAXtVEWjmL5vBOKhR7gfqkDUC96RTgVhGWfKHywOQOWJ6cP+B9DT4esa178"
            crossorigin="anonymous"></script>
    <script>
      SwaggerUIBundle({ url: '/openapi.yaml', dom_id: '#ui' })
    </script>
  </body>
</html>`

func (s *Server) openAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(openAPISpec)
}

func (s *Server) docs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(swaggerPage))
}
