package main

import (
	"errors"
	"sync"
	"time"
)

type Session struct {
	sync.Mutex
	//	maxlifetime int64
	sid    string
	uptime time.Time

	values map[interface{}]interface{}
}

type SessionInterface interface {
	Cb(fn func(s *Session, args ...interface{}) error, args ...interface{}) error
	Get(key interface{}) interface{}
	Id() string
	Set(kkey, value interface{}) error
}

func NewSession(sid string) (sess *Session) {
	sess = &Session{
		sid:    sid,
		values: make(map[interface{}]interface{}),
	}

	sess.up()

	return
}

func (this *Session) Delete(key interface{}) error {
	if key == nil {
		return errors.New("Key must be valid not nil")
	}

	this.Lock()
	this.delete(key)
	this.Unlock()

	return nil
}

func (this *Session) Id() string {
	return this.sid
}

func (this *Session) Cb(fn func(s *Session, args ...interface{}) error, args ...interface{}) (err error) {
	this.Lock()
	err = fn(this, args...)
	this.Unlock()

	return
}

func (this *Session) Get(key interface{}) (value interface{}) {
	this.Lock()
	value = this.get(key)
	this.Unlock()

	return
}

func (this *Session) Set(key, value interface{}) error {
	if key == nil {
		return errors.New("Key and Value must be valid not nil")
	}

	this.Lock()
	this.set(key, value)
	this.Unlock()

	return nil
}

func (this *Session) delete(key interface{}) {
	if _, ok := this.values[key]; ok {
		delete(this.values, key)
	}
}

func (this *Session) get(key interface{}) (value interface{}) {
	value, _ = this.values[key]
	return
}

func (this *Session) set(key, value interface{}) {
	this.values[key] = value
}

func (this *Session) up() {
	this.uptime = time.Now()
}
