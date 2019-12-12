package register

import (
	"network/reactor"
	"reflect"
	"sync"
)

type Factory struct {
	sessionType reflect.Type
}

var onceDefFactory sync.Once
var DefFactory *Factory

func RegisterSession(sessionCls interface{}) {
	onceDefFactory.Do(func() {
		DefFactory = &Factory{
			sessionType: reflect.ValueOf(sessionCls).Type(),
		}
	})
}

func (f *Factory) CreateSession() reactor.Handler {
	return reflect.New(f.sessionType).Interface().(reactor.Handler)
}
