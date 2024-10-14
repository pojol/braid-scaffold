package errcode

import "github.com/pojol/braid/lib/errcode"

var (
	Unknow = func(args ...interface{}) errcode.Code { return errcode.Add(-1, " unknow error", args...) }
)
