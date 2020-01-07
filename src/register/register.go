package register

import (
	"network/reactor"
	"reflect"
	"sync"
)

type Factory struct {
	sessionType reflect.Type
	translator  reactor.Translator
}

var onceDefFactory sync.Once
var DefFactory *Factory

func RegisterFactory(sessionCls interface{}, translator reactor.Translator) {
	onceDefFactory.Do(func() {
		DefFactory = &Factory{
			sessionType: reflect.ValueOf(sessionCls).Type(),
			translator:  translator,
		}
	})
}

func (f *Factory) GetTranslator() reactor.Translator {
	return f.translator
}

func (f *Factory) CreateSession() reactor.Handler {
	return reflect.New(f.sessionType).Interface().(reactor.Handler)
}
