package errcode

import "github.com/pojol/braid/lib/errcode"

var (
	Unknow = func(args ...interface{}) errcode.Code { return errcode.Add(-1, " unknow error", args...) }

	NameLegalErr = func(args ...interface{}) errcode.Code { return errcode.Add(2000, " 用户名不符合规范", args...) } // 用户名不符合规范

)
