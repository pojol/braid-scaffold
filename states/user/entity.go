package user

import (
	commproto "braid-scaffold/states/commproto"
	"braid-scaffold/states/loader"
	"context"
	"reflect"
	"time"

	"github.com/pojol/braid/core"
)

type EntityWrapper struct {
	ID       string              `bson:"_id"`
	cs       core.ICacheStrategy `bson:"-"`
	Bag      *BagModule          `bson:"bag"`
	User     *UserModule         `bson:"user"`
	TimeInfo *TimeInfoModule     `bson:"time_info"`

	// Used to determine if it was read from cache
	isCache bool `bson:"-"`
}

func (e *EntityWrapper) GetID() string {
	return e.ID
}

func (e *EntityWrapper) SetModule(moduleType reflect.Type, module interface{}) {
	switch moduleType {
	case reflect.TypeOf(&BagModule{}):
		e.Bag = module.(*BagModule)
	case reflect.TypeOf(&UserModule{}):
		e.User = module.(*UserModule)
	case reflect.TypeOf(&TimeInfoModule{}):
		e.TimeInfo = module.(*TimeInfoModule)
	}
}

func (e *EntityWrapper) GetModule(moduleType reflect.Type) interface{} {
	switch moduleType {
	case reflect.TypeOf(&BagModule{}):
		return e.Bag
	case reflect.TypeOf(&UserModule{}):
		return e.User
	case reflect.TypeOf(&TimeInfoModule{}):
		return e.TimeInfo
	}
	return nil
}

func NewEntityWapper(id string) *EntityWrapper {
	// 注: loader 需要将 module 指针传入用于建立引用关系，所以这边的 module 需要默认构建出来
	e := &EntityWrapper{
		ID: id,
		User: &UserModule{
			ID: id,
		},
		TimeInfo: &TimeInfoModule{ID: id},
		Bag:      &BagModule{ID: id, Bag: make(map[int32]*commproto.ItemList)},
	}
	e.cs = loader.BuildUserLoader(e)
	return e
}

func (e *EntityWrapper) Load(ctx context.Context) error {
	err := e.cs.Load(ctx)
	if err != nil {
		return err
	}

	e.isCache = true
	e.TimeInfo.SyncTime = time.Now().Unix()

	return nil
}

func (e *EntityWrapper) IsExist() bool {
	return e.cs.IsExist(context.TODO())
}

func (e *EntityWrapper) Sync(ctx context.Context, forceUpdate bool) error {
	return e.cs.Sync(ctx, forceUpdate)
}

func (e *EntityWrapper) Store(ctx context.Context) error {
	return e.cs.Store(ctx)
}

func (e *EntityWrapper) IsDirty() bool {
	return e.cs.IsDirty()
}
