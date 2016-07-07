package main

import (
	"database/sql"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func init() {
	log = NewLogger(1000)
	// Set console log as default
	log.SetLogger("console", `{"level":7}`)
}

func main() {
	var (
		cfg      *Config
		db       *sql.DB
		sessions *Provider
		sig      chan os.Signal
		err      error
	)

	// Read flags
	flag.Parse()

	// Print version and exit
	if PrintVersion {
		showVersion()
	}

	// Read configuration
	if cfg, err = NewConfig(CONFIGFILE); err != nil {
		log.Critical(err.Error())
	} else {
		if err = cfg.Parse(); err != nil {
			log.Critical(err.Error())
		}
	}

	// Prepare statement
	if db, err = openDB(cfg.Database.DSN()); err != nil {
		log.Critical(err.Error())
	}

	// Create sessions storage
	if sessions, err = NewManager(db, 0); err != nil {
		log.Critical(err.Error())
	}

	// Catch system signal to save sessions
	// Close DB connection and flush log
	sig = make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go destruct(sig, sessions, db, log)
	// Run garbage collector
	sessions.GC(0)

	http.HandleFunc("/", NewContext(handleRoot, sessions))

	http.ListenAndServe(cfg.Server, nil)
}

type Context struct {
	db *sql.DB
	r  *http.Request
	s  *Session
}

func NewContext(fn func(http.ResponseWriter, *Context), prov *Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			ctx          *Context
			session, err = prov.Start(w, r)
		)

		if err != nil {
			log.Error(err)
			http.Error(w, "Cannot start session", 500)
			return
		}

		ctx = &Context{
			db: prov.conn,
			r:  r,
			s:  session,
		}

		fn(w, ctx)
	}
}

func handleRoot(w http.ResponseWriter, ctx *Context) {
	ctx.s.Set("up", "tralala")
}

// Create database table
func dbTablePrepare(db *sql.DB) error {
	_, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS `msm_session`(" +
			"`id` varchar(255), " +
			"`started` int, " +
			"`updated` int, " +
			"`data` blob, " +
			"PRIMARY KEY(`id`)" +
			") Engine=MyISAM",
	)

	return err
}

// Exit handler
func destruct(c chan os.Signal, args ...interface{}) {
	<-c

	for _, item := range args {
		switch item.(type) {
		case *Provider:
			item.(*Provider).Flush()
		case *Log:
			item.(*Log).Close()
		case *sql.DB:
			item.(*sql.DB).Close()
		}
	}

	os.Exit(1)
}
