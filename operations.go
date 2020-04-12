package go_sdk

import (
	"reflect"
)

type UpdatePart struct {
	IgnoreError    bool        `json:"ignoreError,omitempty"`
	NoMoreContents bool        `json:"noMoreContents,omitempty"`
	Ops            []Operation `json:"ops,omitempty"`
}

func NewUpdatePart() *UpdatePart {
	return &UpdatePart{}
}

// AddOpSet 添加一个 set 操作，set 操作可以用于设置新值或替换旧值
func (ops *UpdatePart) AddOpSet(set *Set) {
	if set == nil {
		return
	}

	ops.Ops = append(ops.Ops, Operation{Set: set})
}

// AddOpUnset 添加一个 unset 操作，unset 操作用于将旧值从 data 中删除
func (ops *UpdatePart) AddOpUnset(unset *Unset) {
	if unset == nil {
		return
	}

	ops.Ops = append(ops.Ops, Operation{Unset: unset})
}

// AddOpInsert 添加一个 insert 操作，insert 操作可以往已存在的数组中的任意有效位置添加新的值进去
func (ops *UpdatePart) AddOpInsert(insert *Insert) {
	if insert == nil {
		return
	}

	ops.Ops = append(ops.Ops, Operation{Insert: insert})
}

// AddOpRemove 添加一个 remove 操作，remove 操作与 insert 相反，删除数组中指定索引的地方，删除后数组长度变小
func (ops *UpdatePart) AddOpRemove(remove *Remove) {
	if remove == nil {
		return
	}

	ops.Ops = append(ops.Ops, Operation{Remove: remove})
}

// MarkNoMoreContents 设置 noMoreContents 为 true，在上滑滚动翻页操作中，如果已经翻页到尽头了，返回 noMoreContents
// APP 将不会再发起请求，并通知用户已经没有更多数据了
func (ops *UpdatePart) MarkNoMoreContents() {
	ops.NoMoreContents = true
}

type Operation struct {
	Set    *Set    `json:"$set,omitempty"`
	Unset  *Unset  `json:"$unset,omitempty"`
	Insert *Insert `json:"$insert,omitempty"`
	Remove *Remove `json:"$remove,omitempty"`
}

type Set map[string]interface{}

func NewSet() *Set {
	return &Set{}
}

func (op *Set) Add(keyPath string, value interface{}) *Set {
	(*op)[keyPath] = value
	return op
}

func (op *Set) Del(keyPath string) *Set {
	delete(*op, keyPath)
	return op
}

type Unset []string

func NewUnset() *Unset {
	return &Unset{}
}

func (op *Unset) Add(keyPaths ...string) *Unset {
	*op = append(*op, keyPaths...)
	return op
}

func (op *Unset) Del(keyPath string) *Unset {
	var arr = *op
	for i := range arr {
		if arr[i] == keyPath {
			*op = append(arr[:i], arr[i:]...)
			return op
		}
	}

	return op
}

type Insert struct {
	KeyPath string        `json:"$keypath"`
	Ele     []interface{} `json:"$ele"`
	Index   int           `json:"$index"`
}

// NewInsert 生成一个 insert 操作，list 必须是一个数组
func NewInsert(keyPath string, list interface{}) *Insert {
	if list == nil {
		return nil
	}

	rv := reflect.ValueOf(list)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return nil
	}

	size := rv.Len()
	ele := make([]interface{}, size)
	for i := 0; i < size; i++ {
		ele[i] = rv.Index(i).Interface()
	}

	return &Insert{
		KeyPath: keyPath,
		Ele:     ele,
		Index:   -1,
	}
}

type Remove struct {
	KeyPath string `json:"$keypath"`
	Indexes []int  `json:"$indexes"`
}

func NewRemove(keyPath string, indexes []int) *Remove {
	return &Remove{
		KeyPath: keyPath,
		Indexes: indexes,
	}
}
