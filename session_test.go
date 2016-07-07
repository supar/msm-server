package main

import (
	"database/sql"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"testing"
	"time"
)

type SomeSessMock struct {
	User  string
	Value int
}

func InitDBMock(t *testing.T) (db *sql.DB, mock sqlmock.Sqlmock) {
	var (
		err error
	)

	// open database stub
	db, mock, err = sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}

	return
}

func Test_EndodeDecodeGob(t *testing.T) {
	var (
		mock = []map[interface{}]interface{}{
			{
				"anyuser": "gooduser",
				12:        12,
			},
			{
				12:     1247272,
				"user": SomeSessMock{"superuser", 9827643},
			},
		}
	)

	for _, v := range mock {
		b, err := EncodeGob(v)

		if err != nil {
			t.Errorf("Unexpected error: %s", err.Error())
		}

		c, err := DecodeGob(b)

		if err != nil {
			t.Errorf("Unexpected error: %s", err.Error())
		}

		if ok := reflect.DeepEqual(v, c); !ok {
			t.Errorf("Decoded object is not equal original")
		}
	}
}

func Test_ProviderReadExisting(t *testing.T) {
	var (
		err error

		db, mock = InitDBMock(t)

		sessions = []map[interface{}]interface{}{
			{
				"key":  "a7346f123b0",
				"data": SomeSessMock{"anyuser", 387273},
			},
			{
				"key":  "b13cf6f1c3b",
				"data": SomeSessMock{"anyuser", 387273},
			},
		}

		prov, _ = NewManager(db, 0)
	)

	defer db.Close()

	for _, v := range sessions {
		c, _ := EncodeGob(v)

		mock.ExpectQuery("SELECT").
			WithArgs(v["key"]).
			WillReturnRows(sqlmock.NewRows([]string{"data"}).AddRow(c))
	}

	for _, v := range sessions {
		if _, err = prov.read(v["key"].(string)); err != nil {
			t.Error(err)
		}
	}

	// we make sure that all expectations were met
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expections: %s", err.Error())
	}
}

func Test_CreateCookieSidIfCookieNotPasswedInRequest(t *testing.T) {
	var (
		err error

		db, mock = InitDBMock(t)
		prov, _  = NewManager(db, 0)
	)

	mock.ExpectQuery("SELECT").WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"data"}))
	mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/", nil)

	_, err = prov.Start(w, r)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	if ok, _ := regexp.MatchString("msm-server-sid=.", r.Header.Get("Cookie")); !ok {
		t.Errorf("Expexted valid cookie in the response")
	}

	// we make sure that all expectations were met
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expections: %s", err.Error())
	}
}

func Test_ProviderReadAndCreateNew(t *testing.T) {
	var (
		err error

		db, mock = InitDBMock(t)

		sessions = []map[interface{}]interface{}{
			{
				"key":  "exists",
				"data": SomeSessMock{"anyuser", 387273},
			},
			{
				"key":  "notexists",
				"data": SomeSessMock{"anyuser", 387273},
			},
		}

		prov, _ = NewManager(db, 0)
	)

	defer db.Close()

	for _, v := range sessions {
		c, _ := EncodeGob(v)

		if v["key"] == "exists" {
			mock.ExpectQuery("SELECT").
				WithArgs(v["key"]).
				WillReturnRows(sqlmock.NewRows([]string{"data"}).AddRow(c))
		} else {
			mock.ExpectQuery("SELECT").
				WithArgs(v["key"]).WillReturnRows(sqlmock.NewRows([]string{"data"}))

			mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
		}
	}

	for _, v := range sessions {
		if _, err = prov.read(v["key"].(string)); err != nil {
			t.Error(err)
		}
	}

	// we make sure that all expectations were met
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expections: %s", err.Error())
	}
}

func Test_StartSessionWithExistingCookie(t *testing.T) {
	var (
		err  error
		sess *Session

		db, mock = InitDBMock(t)
		prov, _  = NewManager(db, 0)
		sid      = RandStringId(64)
	)

	defer db.Close()

	c, _ := EncodeGob(map[interface{}]interface{}{"data": "somedata"})
	mock.ExpectQuery("SELECT").WithArgs(sid).
		WillReturnRows(sqlmock.NewRows([]string{"data"}).AddRow(c))

	w := httptest.NewRecorder()
	http.SetCookie(w, &http.Cookie{Name: prov.cookieName, Value: sid})

	r, _ := http.NewRequest("GET", "/", nil)
	r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

	sess, err = prov.Start(w, r)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}

	if sess == nil {
		t.Errorf("Expected session object, but got nil")
	}

	// we make sure that all expectations were met
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expections: %s", err.Error())
	}
}

func Test_GarbageRecordsRemove(t *testing.T) {
	var (
		err error

		db, mock = InitDBMock(t)
		prov, _  = NewManager(db, 2)
	)

	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 0))

	if err = prov.garbage(); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	// we make sure that all expectations were met
	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expections: %s", err.Error())
	}
}

func Test_SessionStoreItemAdd(t *testing.T) {
	var (
		queue = make([]string, 20)
		prov  = &Provider{
			store: make([]*Session, 0),
		}
	)

	for i, _ := range queue {
		queue[i] = RandStringId(64)
		prov.append(NewSession(queue[i]))
	}

	if len(prov.store) != len(queue) {
		t.Errorf("Expected equal length")
	}
}

func Test_SessionStoreItemDelete(t *testing.T) {
	var (
		queue = make([]string, 20)
		prov  = &Provider{
			store: make([]*Session, 0),
		}
	)

	for i, _ := range queue {
		queue[i] = RandStringId(64)
		prov.append(NewSession(queue[i]))
	}

	for i, sid := range queue {
		if (i % 2) == 0 {
			idx, _ := prov.get(sid)
			prov.delete(idx)

			if k, _ := prov.get(sid); k != -1 {
				t.Errorf("Unexpected session with id %s", sid)
			}
		}
	}

	if l := len(prov.store); l != len(queue)/2 {
		t.Errorf("Expected store length %d, got %d", len(queue)/2, l)
	}
}

func Test_SessionStoreItemExists(t *testing.T) {
	var (
		queue = make([]string, 20)
		prov  = &Provider{
			store: make([]*Session, 0),
		}
	)

	for i, _ := range queue {
		queue[i] = RandStringId(64)
		prov.append(NewSession(queue[i]))
	}

	for i, sid := range queue {
		if (i % 2) == 0 {
			if idx, s := prov.get(sid); s == nil || s.sid != sid {
				t.Errorf("Expected session with id %s, but got value %v with idx %d", sid, s, idx)
			}
		}
	}
}

func Test_EachItemCallbackExecute(t *testing.T) {
	var (
		queue = make([]string, 20)
		prov  = &Provider{
			store: make([]*Session, 0),
		}
	)

	for i, _ := range queue {
		queue[i] = RandStringId(64)
		prov.append(NewSession(queue[i]))
	}

	l := len(queue)
	prov.each(func(idx int, s *Session) bool {
		if idx < 0 || idx >= l {
			t.Fatalf("Unexpected index value %d", idx)
		}

		return true
	})
}

func Test_KeepAliveSessionsInTheCache(t *testing.T) {
	var (
		db, _   = InitDBMock(t)
		prov, _ = NewManager(db, 1)
		queue   = 20
	)

	for i := 0; i < queue; i++ {
		s := NewSession(RandStringId(64))

		if (i % 2) == 0 {
			s.uptime = s.uptime.Add(-10 * time.Duration(60) * time.Second)
		}

		prov.append(s)
	}

	prov.keepAlive()

	if l := len(prov.store); l != (queue / 2) {
		t.Errorf("Expected storage length %d, but got %d", (queue / 2), l)
	}
}

func Test_SessionConcuranceCallback(t *testing.T) {
	var (
		sess = NewSession("827364g3656g")

		iter  = make(chan int)
		done  = make(chan bool)
		queue = 100
	)

	// Create routine to triiger the end of asynchronous writing
	go func() {
		defer func() { done <- true }()
		var cover int

		for {
			select {
			case i := <-iter:
				cover += i

				if cover >= queue {
					return
				}
			}
		}
	}()

	// Create attempts for each session
	for a := 0; a < queue; a++ {
		go func(sess SessionInterface, q int) {
			fn := func(s *Session, args ...interface{}) error {
				v := s.get("store")

				switch v.(type) {
				case int64:
					s.set("store", v.(int64)+1)
				default:
					s.set("store", int64(1))
				}

				return nil
			}

			// Let's get and set store item long time
			for k := 0; k < 10000; k++ {
				sess.Cb(fn)
			}

			iter <- 1
		}(sess, a)
	}

	<-done

	if v := sess.Get("store"); v == nil || v.(int64) != 1000000 {
		t.Errorf("Expected value %d at %s, but got %d", 1000000, sess.sid, v.(int64))
	}
}

func Test_StartSessionConcurrance(t *testing.T) {
	var (
		sess *Session
		err  error

		iter     = make(chan int)
		done     = make(chan bool)
		db, mock = InitDBMock(t)
		prov, _  = NewManager(db, 0)
		queue    = make([]string, 20)
	)

	prov.cacheTimer.Reset(time.Second / 3)
	defer db.Close()

	// Create routine to triiger the end of asynchronous writing
	go func() {
		defer func() { done <- true }()
		var cover int

		for {
			select {
			case i := <-iter:
				cover += i

				if cover >= len(queue)/2 {
					return
				}
			}
		}
	}()

	// Simulate http request with session cookie
	// Http request handler
	handler := func(prov ProviderInterface, sid string) (*Session, error) {
		w := httptest.NewRecorder()
		http.SetCookie(w, &http.Cookie{Name: prov.Name(), Value: sid})

		r, _ := http.NewRequest("GET", "/", nil)
		r.Header.Set("Cookie", w.Header().Get("Set-Cookie"))

		return prov.Start(w, r)
	}

	// Create synchronuos requests
	for i, _ := range queue {
		queue[i] = RandStringId(64)

		c, _ := EncodeGob(map[interface{}]interface{}{"siddata": queue[i]})
		mock.ExpectQuery("SELECT").WithArgs(queue[i]).
			WillReturnRows(sqlmock.NewRows([]string{"data"}).AddRow(c))

		sess, err = handler(prov, queue[i])

		if (i % 1) == 0 {
			sess.uptime = sess.uptime.Add(-1 * 121 * prov.cacheLifeTime)
		}

		if err != nil {
			t.Fatalf("Unexpected error: %s", err.Error())
		}

		if sess == nil {
			t.Errorf("Expected session object, but got nil")
		}
	}

	if l := len(prov.store); l != len(queue) {
		t.Fatalf("Expected provider store length %d, but got %d", len(queue), l)
	}

	// Create request with concurrency
	concurrent := func(prov ProviderInterface, sid string, fn func(ProviderInterface, string) (*Session, error)) {
		defer func() { iter <- 1 }()

		for a := 0; a < 100; a++ {
			if s, _ := fn(prov, sid); s != nil {
				for b := 0; b < 10000; b++ {
					s.Set("store", b)
				}
			}
		}
	}

	for i, sid := range queue {
		if (i % 2) == 0 {
			go concurrent(prov, sid, handler)
		}
	}

	<-done

	// we make sure that all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("There were unfulfilled expections: %s", err.Error())
	}
}
