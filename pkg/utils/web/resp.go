package web

import "fmt"

const (
	success = "success"
)

// State 基础响应结构体
type State struct {
	// state 状态码，0表示成功，其他表示失败
	Code int `json:"code"`
	// state 具体的信息，成功时为success，失败时为具体的错误信息
	Msg string `json:"msg"`
}

// Response 响应结构体
type Response struct {
	State State `json:"state"`
	// 如果有响应数据，单一数据则直接返回，多个数据则返回数组结构
	Data interface{} `json:"data,omitempty"`
}

// ExceptResponse 异常响应
func ExceptResponse(code int, msg ...interface{}) *Response {
	return &Response{
		State: State{
			Code: code,
			Msg:  fmt.Sprint(msg...),
		},
	}
}

// DataResponse 单一数据响应
func DataResponse(data interface{}) *Response {
	return &Response{
		State: State{
			Code: 0,
			Msg:  success,
		},
		Data: data,
	}
}

// ListData 列表数据响应基础结构
type ListData struct {
	Items interface{} `json:"items,omitempty"`
	Total int         `json:"total"`
}

// ListResponse 基础结构响应
func ListResponse(total int, data interface{}) *Response {
	return &Response{
		State: State{
			Code: 0,
			Msg:  success,
		},
		Data: ListData{
			Items: data,
			Total: total,
		},
	}
}

// OkResponse 无数据的成功响应
func OkResponse() *Response {
	return &Response{
		State: State{
			Code: 0,
			Msg:  success,
		},
	}
}
