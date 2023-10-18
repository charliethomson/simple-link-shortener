package main

import (
	"github.com/charliethomson/link-shortener/database"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/time/rate"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
func main() {

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(
		rate.Limit(20),
	)))

	t := &Template{
		templates: template.Must(template.ParseGlob("public/*.html")),
	}
	e.Renderer = t

	// Routes
	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index", nil)
	})
	e.GET("/:slug", resolveLink)
	e.GET("/links/:linkId", linkCard)
	e.POST("/links", createLink)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}

var validRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789$-_.+!*'()")

func randomString(length int) string {
	bytes := make([]rune, length)
	for i := range bytes {
		bytes[i] = validRunes[rand.Intn(len(validRunes))]
	}
	return string(bytes)
}

type Link struct {
	Id        int    `db:"id"`
	ShortUrl  string `db:"short_url"`
	LongUrl   string `db:"long_url"`
	CreatedAt string `db:"created_at"`
}

func resolveLink(c echo.Context) error {
	connection, err := database.GetConnection()
	if err != nil {
		log.Fatalln("Failed to get database connection")
	}
	slug := c.Param("slug")
	link := Link{}
	if err := connection.Get(&link, "select * from links where links.short_url = ?", slug); err != nil {
		log.Printf("Select error: %s", err.Error())
		return c.Render(http.StatusNotFound, "not-found", nil)
	}
	_, _ = connection.Exec("insert into link_accesses (link_id) values (?)", link.Id)
	return c.Redirect(http.StatusPermanentRedirect, link.LongUrl)
}

type CreateLinkQuery struct {
	Url       string  `json:"url" form:"url"`
	CustomUrl *string `json:"customUrl" form:"customUrl"`
	MaxLength *int    `json:"maxLength" form:"maxLength"`
}

func generateUrl(query *CreateLinkQuery) string {
	if query.CustomUrl != nil && len(*query.CustomUrl) >= 3 {
		return *query.CustomUrl
	}
	if query.MaxLength == nil {
		log.Fatalln("Bad request :sob:")
	}
	return randomString(*query.MaxLength)
}

func createLink(c echo.Context) error {
	q := CreateLinkQuery{}
	if err := c.Bind(&q); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if q.MaxLength != nil && *q.MaxLength > 0 && *q.MaxLength < 3 {
		return echo.NewHTTPError(http.StatusBadRequest, "maxLength must be greater than 3")
	}
	connection, err := database.GetConnection()

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	url := generateUrl(&q)
	result, err := connection.Exec("insert into links (long_url, short_url, created_at) values (?, ?, CURRENT_TIMESTAMP)", q.Url, url)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	newLinkId, err := result.LastInsertId()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	newLink := Link{}
	if err := connection.Get(&newLink, "select * from links where links.id = ?", newLinkId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	type CreatedLinkParams struct {
		Link      Link
		Id        int
		ShortUrl  string
		LongUrl   string
		CreatedAt string
	}
	return c.Render(http.StatusOK, "created-link", CreatedLinkParams{
		Link:      newLink,
		Id:        newLink.Id,
		ShortUrl:  newLink.ShortUrl,
		LongUrl:   newLink.LongUrl,
		CreatedAt: newLink.CreatedAt,
	})
}

func linkCard(c echo.Context) error {
	connection, err := database.GetConnection()

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	linkId, err := strconv.Atoi(c.Param("linkId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	newLink := Link{}
	if err := connection.Get(&newLink, "select * from links where links.id = ?", linkId); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.Render(http.StatusOK, "link", newLink)
}
