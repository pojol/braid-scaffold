package loader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	trhreids "github.com/pojol/braid/3rd/redis"
	"github.com/pojol/braid/core"
	"github.com/pojol/braid/lib/log"
	"github.com/redis/go-redis/v9"
)

type BlockLoader struct {
	BlockName string
	BlockType reflect.Type

	Ins      interface{}
	oldBytes []byte
}

type UserCacheLoader struct {
	WrapperEntity core.IEntity
	Loaders       []BlockLoader
}

var (
	ErrEmptyUser = errors.New("empty user")
)

func BuildUserLoader(wrapper core.IEntity) *UserCacheLoader {
	wrapperType := reflect.TypeOf(wrapper).Elem()
	wrapperValue := reflect.ValueOf(wrapper).Elem()
	loaders := make([]BlockLoader, 0)

	for i := 0; i < wrapperType.NumField(); i++ {
		field := wrapperType.Field(i)
		fieldValue := wrapperValue.Field(i)
		if field.Type.Kind() == reflect.Ptr {
			elemType := field.Type.Elem()
			if elemType.Kind() == reflect.Struct || (elemType.Kind() == reflect.Slice && elemType.Elem().Kind() == reflect.Struct) {
				bsonTag := field.Tag.Get("bson")
				blockName := strings.Split(bsonTag, ",")[0] // 获取 bson 标签的第一部分作为名称
				if blockName == "" {
					blockName = strings.ToLower(field.Name) // 如果没有 bson 标签，使用字段名的小写形式
				}
				loaders = append(loaders, BlockLoader{
					BlockName: blockName,
					BlockType: field.Type,
					Ins:       fieldValue.Interface(),
				})
			}
		}
	}
	return &UserCacheLoader{WrapperEntity: wrapper, Loaders: loaders}
}

func (loader *UserCacheLoader) Load(ctx context.Context) error {
	if len(loader.Loaders) == 0 {
		return fmt.Errorf("loader user empty loaders %v", loader.WrapperEntity.GetID())
	}

	var cmds []redis.Cmder

	cmds, err := trhreids.TxPipelined(ctx, "[EntityLoader.Load]", func(pipe redis.Pipeliner) error {
		for _, load := range loader.Loaders {
			key := fmt.Sprintf("entity_{%s}_%s", loader.WrapperEntity.GetID(), load.BlockName)
			pipe.Get(ctx, key)
		}
		return nil
	})
	if err != nil {
		if err == redis.Nil {
			return ErrEmptyUser
		} else {
			return err
		}
	}

	var bytSlice [][]byte
	bytSlice, err = trhreids.GetCmdsByteSlice(cmds)
	if err != nil {
		return err
	}

	for idx, load := range loader.Loaders {
		protoMsg := reflect.New(load.BlockType.Elem()).Interface().(proto.Message)

		if len(bytSlice[idx]) == 0 {
			return fmt.Errorf("load block %s is not empty", load.BlockName)
		}

		if err := proto.Unmarshal(bytSlice[idx], protoMsg); err != nil {
			return fmt.Errorf("loader unmarshal err %v %v", loader.WrapperEntity.GetID(), load.BlockName)
		}

		// init nil pointer
		initNestedPointers(reflect.ValueOf(protoMsg).Elem())

		loader.Loaders[idx].oldBytes = bytSlice[idx]
		loader.Loaders[idx].Ins = protoMsg
		loader.WrapperEntity.SetModule(load.BlockType, protoMsg)
	}

	return nil
}

func initNestedPointers(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Kind() == reflect.Ptr && field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		if field.Kind() == reflect.Ptr {
			initNestedPointers(field.Elem())
		} else if field.Kind() == reflect.Struct {
			initNestedPointers(field)
		}
	}
}

func (loader *UserCacheLoader) Sync(ctx context.Context, forceUpdate bool) error {
	if len(loader.Loaders) == 0 {
		return fmt.Errorf("loader user empty loaders %v", loader.WrapperEntity.GetID())
	}

	_, err := trhreids.TxPipelined(ctx, "[EntityLoader.Sync]", func(pipe redis.Pipeliner) error {
		for idx, load := range loader.Loaders {
			if loader.Loaders[idx].Ins == nil {
				log.WarnF("sync %s Ins is nil", load.BlockName)
				continue
			}

			protoMsg, ok := loader.Loaders[idx].Ins.(proto.Message)
			if !ok {
				log.WarnF("module %s does not implement proto.Message, skipping", load.BlockName)
				continue
			}

			// 添加额外的 nil 检查
			if reflect.ValueOf(protoMsg).IsNil() {
				log.WarnF("module %s is nil, skipping", load.BlockName)
				fmt.Println(loader.Loaders[idx].Ins)
				continue
			}

			byt, err := proto.Marshal(protoMsg)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", loader.Loaders[idx].BlockName, err)
			}

			if forceUpdate || !bytes.Equal(loader.Loaders[idx].oldBytes, byt) {
				loader.Loaders[idx].oldBytes = byt // update
				key := fmt.Sprintf("entity_{%s}_%s", loader.WrapperEntity.GetID(), load.BlockName)
				pipe.Set(ctx, key, byt, 0)
			}
		}
		return nil
	})

	return err
}

func (loader *UserCacheLoader) Store(ctx context.Context) error {
	return nil
}

func (loader *UserCacheLoader) IsDirty() bool {
	for _, load := range loader.Loaders {

		byt, err := proto.Marshal(load.Ins.(proto.Message))
		if err != nil {
			return false
		}

		if !bytes.Equal(load.oldBytes, byt) {
			return true
		}
	}

	return false
}

func (loader *UserCacheLoader) IsExist(ctx context.Context) bool {
	key := fmt.Sprintf("entity_{%s}_user", loader.WrapperEntity.GetID())

	existsCmd := trhreids.Exists(ctx, key)
	exists, err := existsCmd.Result()
	if err != nil {
		log.ErrorF("Error checking if user exists: %v", err)
		return false
	}

	return exists > 0
}
