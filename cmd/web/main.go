package main

import (
	"database/sql"
	"expvar"
	"fmt"
	"runtime"
	"strconv"

	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Backblaze/blazer/b2"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"

	// "github.com/go-playground/form"
	_ "net/http/pprof"

	"github.com/go-playground/form/v4"
	_ "github.com/lib/pq"
	"zeeshan.kineta.site/internal/mailer"
	"zeeshan.kineta.site/internal/model"
)

type config struct {
	port string
	//env  string
	db struct {
		dsn          string
		maxOpenConne int
		maxIdleConn  int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     int
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
	supabase struct {
		Supabase_Project_url string
		Supabase_Service_key string
		Subase_Storage       string
	}
}

type Application struct {
	Config         *config
	Posts          *model.PostModel
	Tokens         *model.TokenModel
	sessionmanager *scs.SessionManager
	UserModel      *model.UserModel
	Infolog        *log.Logger
	Errorlog       *log.Logger
	Decoder        *form.Decoder
	TempCache      map[string]*template.Template
	Mailer         mailer.Mailer
	Bucket         *b2.Bucket
}

const version = "1.0.0"

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	var cfg config

	// Port configuration
	if cfg.port = os.Getenv("PORT"); cfg.port == "" {
		cfg.port = ":8080"
	}
	flag.StringVar(&cfg.port, "addr", cfg.port, "ADDR")

	// Database configuration
	if cfg.db.dsn = os.Getenv("DB_DSNS"); cfg.db.dsn == "" {
		flag.StringVar(&cfg.db.dsn, "dsn", "", "DSN")
	} else {
		flag.StringVar(&cfg.db.dsn, "dsn", cfg.db.dsn, "DSN")
	}

	fmt.Println("db dns -> ", cfg.db.dsn)

	// Parse ints from env with defaults
	maxOpenConn, _ := strconv.Atoi(os.Getenv("MAX_OPEN_CONN"))
	if maxOpenConn == 0 {
		maxOpenConn = 60
	}
	flag.IntVar(&cfg.db.maxOpenConne, "max-open-con", maxOpenConn, "open connections")

	maxIdleConn, _ := strconv.Atoi(os.Getenv("MAX_IDLE_CONN"))
	if maxIdleConn == 0 {
		maxIdleConn = 20
	}
	flag.IntVar(&cfg.db.maxIdleConn, "max-idle-con", maxIdleConn, "max idle connections")

	// Parse duration from env
	maxIdleTime := os.Getenv("MAX_IDLE_TIME")
	if maxIdleTime == "" {
		maxIdleTime = "5m"
	}
	idleTime, _ := time.ParseDuration(maxIdleTime)
	flag.DurationVar(&cfg.db.maxIdleTime, "max-idle-time", idleTime, "max idle timeout")

	// Rate limit configuration
	limiterRPS, _ := strconv.Atoi(os.Getenv("LIMITER_RPS"))
	if limiterRPS == 0 {
		limiterRPS = 2
	}
	flag.IntVar(&cfg.limiter.rps, "limiter-rps", limiterRPS, "limiter req per sec")

	limiterBurst, _ := strconv.Atoi(os.Getenv("LIMITER_BURST"))
	if limiterBurst == 0 {
		limiterBurst = 3
	}
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", limiterBurst, "limiter burst of reqs")

	limiterEnabled := os.Getenv("LIMITER_ENABLED")
	enabled := true
	if limiterEnabled == "false" {
		enabled = false
	}
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", enabled, "limiter enabling")

	// SMTP configuration - all from env now
	cfg.smtp.host = os.Getenv("SMTP_HOST")
	if cfg.smtp.host == "" {
		cfg.smtp.host = "sandbox.smtp.mailtrap.io"
	}
	flag.StringVar(&cfg.smtp.host, "smtp-host", cfg.smtp.host, "host for smtp")

	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if smtpPort == 0 {
		smtpPort = 25
	}
	cfg.smtp.port = smtpPort
	flag.IntVar(&cfg.smtp.port, "smtp-port", cfg.smtp.port, "port for smtp")

	cfg.smtp.username = os.Getenv("SMTP_USERNAME")
	fmt.Println("\nthis is username \n-> ", cfg.smtp.username)
	flag.StringVar(&cfg.smtp.username, "smtp-username", cfg.smtp.username, "username for smtp")

	cfg.smtp.password = os.Getenv("SMTP_PASSWORD")

	flag.StringVar(&cfg.smtp.password, "smtp-password", cfg.smtp.password, "password for smtp")

	cfg.smtp.sender = os.Getenv("SMTP_SENDER")
	if cfg.smtp.sender == "" {
		cfg.smtp.sender = "Zeeshannet <no-reply@kineta.site>"
	}
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", cfg.smtp.sender, "smtp sender")

	cfg.supabase.Subase_Storage = os.Getenv("Subase_Storage")
	fmt.Println("\nthis is supabase storage\n -> ", cfg.supabase.Subase_Storage)
	cfg.supabase.Supabase_Project_url = os.Getenv("Supabase_Project_url")
	fmt.Println("\nthis is supabase project url\n -> ", cfg.supabase.Supabase_Project_url)
	cfg.supabase.Supabase_Service_key = os.Getenv("Supabase_Service_key")
	fmt.Println("\nthis is supabase service key\n -> ", cfg.supabase.Supabase_Service_key)

	// Parse flags (command line args will override env vars)
	flag.Parse()

	flag.Func("cors-trusted-origin", "Trusted CORS origin (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	expvar.NewString("version").Set(version)

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	flag.Parse()

	info := log.New(os.Stdout, "INFO\t", log.Ltime|log.Ldate)
	Errorlog := log.New(os.Stderr, "ERROR\t", log.Lshortfile)
	db, err := Opendb(&cfg)
	if err != nil {
		Errorlog.Print(err)
		return
	}
	expvar.Publish("db", expvar.Func(func() any {
		return db.Stats()
	}))
	Decoder := form.NewDecoder()
	NewtempCache, err := NewTemplateCache()
	if err != nil {
		Errorlog.Print(err)
	}
	SessionManager := scs.New()
	SessionManager.Store = postgresstore.New(db)
	SessionManager.Cookie.Secure = true
	SessionManager.Lifetime = 12 * time.Hour

	fmt.Print("ruuned")
	b2 := Bbintialize()

	app := &Application{
		Config:         &cfg,
		sessionmanager: SessionManager,
		Infolog:        info,
		Posts:          &model.PostModel{DB: db},
		Tokens:         &model.TokenModel{DB: db},
		Errorlog:       Errorlog,
		UserModel:      &model.UserModel{DB: db},
		Decoder:        Decoder,
		TempCache:      NewtempCache,
		Mailer:         mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		Bucket:         b2,
	}

	fmt.Print("\n attributes")

	app.seee()
	baseurl := app.BaseUrl()
	name := app.Name()
	fmt.Print("\n baseurl:", baseurl)
	fmt.Print("\n Name :", name)
	srv := &http.Server{
		ErrorLog: Errorlog,
		Handler:  app.Routes(),
		Addr:     cfg.port,
	}

	app.Infolog.Printf("server is listening on %s", cfg.port)

	err = srv.ListenAndServe()
	if err != nil {
		app.Errorlog.Print(err)
		return
	}

}

func Opendb(cfg *config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.db.maxOpenConne)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)
	db.SetMaxIdleConns(cfg.db.maxIdleConn)
	return db, nil
}
