package main

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Provider struct {
	// Seconds. Keep session data from DB in the memmory
	cacheLifeTime time.Duration
	cookieName    string
	conn          *sql.DB
	// Hours. Database garbage collector value
	maxAge int64

	lock sync.Mutex
	// Memmory storage for the active sessions
	store []*Session
}

type ProviderInterface interface {
	Start(http.ResponseWriter, *http.Request) (*Session, error)
	Name() string
}

func init() {
	gob.Register([]interface{}{})
	gob.Register(map[int]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register(map[interface{}]interface{}{})
	gob.Register(map[string]string{})
	gob.Register(map[int]string{})
	gob.Register(map[int]int{})
	gob.Register(map[int]int64{})
}

func NewManager(db *sql.DB, maxlifetime int64) (manager *Provider, err error) {
	if db == nil {
		return nil, errors.New("Valid database connection required")
	}

	manager = &Provider{
		cacheLifeTime: time.Duration(60) * time.Second,
		cookieName:    NAME + "-sid",
		conn:          db,
		maxAge:        168,
		store:         make([]*Session, 0),
	}

	return
}

// Get session from storage
func (this *Provider) get(sid string) (idx int, session *Session) {
	var (
		fn func(int, *Session) bool
	)

	idx = -1

	fn = func(i int, s *Session) bool {
		if s.sid == sid {
			idx = i
			session = s

			return false
		}

		return true
	}

	this.each(fn)

	return
}

func (this *Provider) Flush() {
	this.flushCache()
	time.AfterFunc(this.cacheLifeTime, func() { this.Flush() })
}

func (this *Provider) Name() string {
	return this.cookieName
}

func (this *Provider) Start(w http.ResponseWriter, r *http.Request) (session *Session, err error) {
	var (
		cookie *http.Cookie
		sid    string
	)

	if sid, err = this.getSid(r); err != nil {
		return nil, err
	}

	this.lock.Lock()
	_, session = this.get(sid)
	this.lock.Unlock()

	if session == nil {
		if session, err = this.read(sid); err != nil {
			return nil, err
		}

		this.lock.Lock()
		this.append(session)
		this.lock.Unlock()
	}

	cookie = &http.Cookie{
		Name:     this.cookieName,
		Value:    url.QueryEscape(sid),
		Path:     "/",
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	r.AddCookie(cookie)

	return
}

func (this *Provider) getSid(r *http.Request) (string, error) {
	cookie, err := r.Cookie(this.cookieName)

	if err != nil || cookie.Value == "" || cookie.MaxAge < 0 {
		err := r.ParseForm()
		if err != nil {
			return "", err
		}

		sid := r.FormValue(this.cookieName)
		return sid, nil
	}

	// HTTP Request contains cookie for sessionid info.
	return url.QueryUnescape(cookie.Value)
}

func (this *Provider) flushCache() {
	this.lock.Lock()
	defer this.lock.Unlock()

	var (
	//gc = time.Now().Add(-1 * this.cacheLifeTime)
	)

	this.each(func(i int, s *Session) bool {
		var ok bool

		//s.Lock()
		//gc.After(s.uptime)
		//s.Unlock()

		if ok == true {
			this.save(s)
			this.delete(i)
		}
		return true
	})
}

func (this *Provider) append(session *Session) {
	this.store = append(this.store, session)
}

// Remove slice item by index
func (this *Provider) delete(i int) {
	var l = len(this.store)

	if i > -1 && i < l {
		copy(this.store[i:], this.store[i+1:])
		this.store[l-1] = nil
		this.store = this.store[:l-1]
	}
}

// Restore session from DB or create new if not exists
func (this *Provider) read(sid string) (session *Session, err error) {
	var (
		row         *sql.Row
		sessiondata []byte
	)

	session = NewSession(sid)

	row = this.conn.QueryRow("SELECT `data` FROM `msm_session` WHERE `id` = ?", sid)
	err = row.Scan(&sessiondata)

	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}

		_, err = this.conn.Exec("INSERT INTO `msm_session`(`id`,`data`,`starte) VALUES(?, ?, ?)",
			sid, "", time.Now().Unix())

		if err != nil {
			return nil, err
		}
	}

	if len(sessiondata) > 0 {
		session.values, err = DecodeGob(sessiondata)

		if err != nil {
			return nil, err
		}
	}

	return
}

func (this *Provider) each(callback func(int, *Session) bool) {
	for idx, session := range this.store {
		if callback(idx, session) == false {
			break
		}
	}
}

func (this *Provider) save(s *Session) (err error) {
	var (
		data []byte
	)

	if data, err = EncodeGob(s.values); err != nil {
		return
	}

	_, err = this.conn.Exec("UPDATE `msm_session` SET `data` = ?, `updated` = ? WHERE `id` = ?", data, time.Now().Unix(), s.sid)

	return
}

func EncodeGob(obj map[interface{}]interface{}) ([]byte, error) {
	var (
		buffer  *bytes.Buffer
		encoder *gob.Encoder
	)

	for _, v := range obj {
		gob.Register(v)
	}

	buffer = bytes.NewBuffer(nil)
	encoder = gob.NewEncoder(buffer)

	if err := encoder.Encode(obj); err != nil {
		return []byte(""), err
	}

	return buffer.Bytes(), nil
}

func DecodeGob(encoded []byte) (out map[interface{}]interface{}, err error) {
	var (
		buffer  *bytes.Buffer
		decoder *gob.Decoder
	)

	buffer = bytes.NewBuffer(encoded)
	decoder = gob.NewDecoder(buffer)

	if err = decoder.Decode(&out); err != nil {
		return nil, err
	}

	return
}
