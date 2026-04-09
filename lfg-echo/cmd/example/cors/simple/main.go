package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Simple CORS</h1>
    <div id="output"></div>
    <script>
        document.addEventListener('DOMContentLoaded', function() {
            fetch("http://localhost:4000/v1/healthcheck").then(
                function (response) {
                    response.text().then(function (text) {
                        document.getElementById("output").innerHTML = text;
                    });
                },
                function(err) {
                    document.getElementById("output").innerHTML = err;
                }
            );
        });
    </script>
</body>
</html>
`

func main() {
	addr := flag.String("addr", ":9000", "Server address")
	flag.Parse()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Echo handlers return an error; returning c.HTML writes the response body.
	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, html)
	})

	log.Printf("starting server on %s", *addr)
	log.Fatal(e.Start(*addr))
}
