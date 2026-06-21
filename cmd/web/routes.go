package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"zeeshan.kineta.site/ui"
)

func (app *Application) Routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.pasteout)

	dynamic := alice.New(app.sessionmanager.LoadAndSave, app.Authenticate, app.Activate)
	fileServer := http.FileServer(http.FS(ui.Files))
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)
	protected := dynamic.Append(app.RequireAuth)
	downloadred := protected.Append(app.DownloadRedirecting)
	ForActiveUSers := protected.Append(app.RequireActive)
	router.Handler(http.MethodGet, "/about", dynamic.ThenFunc(app.About))
	router.Handler(http.MethodGet, "/signup", dynamic.ThenFunc(app.Signup))
	router.Handler(http.MethodPost, "/signup", dynamic.ThenFunc(app.SignupPost))
	router.Handler(http.MethodGet, "/check/usersdet", dynamic.ThenFunc(app.CheckingDbStats))
	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.Home))
	router.Handler(http.MethodGet, "/search", dynamic.ThenFunc(app.SearchUserProfile))
	router.Handler(http.MethodGet, "/searchp", dynamic.ThenFunc(app.SearchPost))
	router.Handler(http.MethodGet, "/posts/:id", protected.ThenFunc(app.View))
	router.Handler(http.MethodGet, "/download/:id", downloadred.ThenFunc(app.PdfDownloader))
	router.Handler(http.MethodGet, "/create", ForActiveUSers.ThenFunc(app.PostForm))
	router.Handler(http.MethodPost, "/create", ForActiveUSers.ThenFunc(app.PostPost))
	router.Handler(http.MethodGet, "/premium", ForActiveUSers.ThenFunc(app.Premium))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.login))
	router.Handler(http.MethodGet, "/users/:id", dynamic.ThenFunc(app.PostsByUser))
	router.Handler(http.MethodPost, "/AccountActivation", dynamic.ThenFunc(app.ConfirmEmailPost))
	router.Handler(http.MethodGet, "/confirmemail", dynamic.ThenFunc(app.ActivateAccoun))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.LoginPost))
	router.Handler(http.MethodPost, "/posts/:id", ForActiveUSers.ThenFunc(app.MakeComment))
	router.Handler(http.MethodGet, "/rating/posts/:id", protected.ThenFunc(app.Ratingpage))
	router.Handler(http.MethodPost, "/rating/posts/:id", protected.ThenFunc(app.Rating))
	// to see the matrics
	//router.Handler(http.MethodGet, "/debug/var", expvar.Handler())

	standard := alice.New(app.RecoverPanic, app.SecurityHeader, app.GlobalRateLimiter)

	return standard.Then(router)

}
