package fields

import (
	"github.com/pojol/braid/lib/errcode"
	"github.com/pojol/braid/router/msg"
)

const (
	KeyErrCode   = "ErrCode"
	KeyErrMsg    = "ErrMsg"
	KeyActorID   = "ActorID"
	KeyActorTy   = "ActorTy"
	KeyUserID    = "UserID"
	KeyGateID    = "GateID"
	KeySessionID = "SessionID"
	KeyMutexID   = "MutexID"
)

func ErrCode(code errcode.Code) msg.Attr { return msg.Attr{Key: KeyErrCode, Value: code} }
func ErrMsg(errmsg string) msg.Attr      { return msg.Attr{Key: KeyErrMsg, Value: errmsg} }
func ActorID(id string) msg.Attr         { return msg.Attr{Key: KeyActorID, Value: id} }
func ActorTy(ty string) msg.Attr         { return msg.Attr{Key: KeyActorTy, Value: ty} }
func UserID(id string) msg.Attr          { return msg.Attr{Key: KeyUserID, Value: id} }
func GateID(id string) msg.Attr          { return msg.Attr{Key: KeyGateID, Value: id} }
func SessionID(id string) msg.Attr       { return msg.Attr{Key: KeySessionID, Value: id} }
func MutexID(id string) msg.Attr         { return msg.Attr{Key: KeyMutexID, Value: id} }
