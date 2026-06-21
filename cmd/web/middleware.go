package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/tomasen/realip"
	"golang.org/x/time/rate"
	"zeeshan.kineta.site/internal/model"
)

func (app *Application) SecurityHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self' ; img-src https://eeutgcuqpkmykwdomhzl.supabase.co/  https://i.ibb.co/;")
		w.Header().Set("Reffe-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "no-sniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

func (app *Application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.Infolog.Printf("%s, %s, %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (app *Application) Metrics(next http.Handler) http.Handler {
	var (
		RequestRecived        = expvar.NewInt("request_recieved")
		ResponseSend          = expvar.NewInt("response_send")
		totalProcessingTimeMS = expvar.NewInt("processing_time")
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()

		RequestRecived.Add(1)

		next.ServeHTTP(w, r)

		ResponseSend.Add(1)

		duration := time.Since(start).Microseconds()
		totalProcessingTimeMS.Add(duration)

	})
}

func (app *Application) IpBasedRateLImit(next http.Handler) http.Handler {

	type client struct {
		limiter  *rate.Limiter
		lastseen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastseen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.Config.limiter.enabled {
			ip := realip.FromRequest(r)
			log.Println("ip address", ip)

			mu.Lock()
			if _, found := clients[ip]; !found {
				clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.Config.limiter.rps), app.Config.limiter.burst)}
			}

			clients[ip].lastseen = time.Now()

			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.RateLimitExcedded(w, r)
				return
			}
			mu.Unlock()
		}
		next.ServeHTTP(w, r)

	})
}

//
//
//

func (app *Application) GlobalRateLimiter(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(4, 500)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if app.Config.limiter.enabled {
			if !limiter.Allow() {
				app.RateLimitExcedded(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (app *Application) RecoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("connection", "close")
				app.ServerError(w, fmt.Errorf("%s", err))
				return
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *Application) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}
		err, exists := app.UserModel.Exists(id)
		if err != nil {
			app.Errorlog.Print(err)
			return
		}
		if exists {
			ctx := context.WithValue(r.Context(), IsAuthenticatedContextkey, true)
			r = r.WithContext(ctx)

		}
		next.ServeHTTP(w, r)
	})
}

func (app *Application) DownloadRedirecting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userid := app.sessionmanager.GetInt(r.Context(), "authenticateduserid")
		if userid == 0 {
			http.Redirect(w, r, "/about", http.StatusSeeOther)
			next.ServeHTTP(w, r)
			return
		}
		count, err := app.Posts.ReturnDownloadCounts(userid)

		if err != nil {
			switch {
			case errors.Is(err, model.ErrecordNotFound):
				next.ServeHTTP(w, r)
			default:
				app.ServerError(w, err)
			}
			return
		}
		if count >= 4 {
			http.Redirect(w, r, "/about", http.StatusSeeOther)
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *Application) Activate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activated := app.sessionmanager.GetBool(r.Context(), "isActivated")
		if !activated {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), IsActivatedContextKey, true)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *Application) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.IsAuthenticated(r) {

			http.Redirect(w, r, "/signup", http.StatusSeeOther)
			return
		}
		w.Header().Set("cache-control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (app *Application) RequireActive(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.IsActivated(r) {
			http.Redirect(w, r, "/nigga", http.StatusSeeOther)
			return
		}
		w.Header().Set("cache-control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (app *Application) EnableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Vary", "Origin")
		origin := r.Header.Get("Origin")

		if origin != "" {
			for i := range app.Config.cors.trustedOrigins {
				if origin == app.Config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)

				}
			}
		}
		next.ServeHTTP(w, r)

	})
}
