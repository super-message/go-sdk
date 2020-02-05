package go_sdk

import (
	"fmt"
)

// APIError 定义平台接口统一的返回错误信息的结构
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error[%d]: %s", e.Code, e.Message)
}

// IsInvalidRequestToken 返回错误是否为 request token 无效引起的
func (e *APIError) IsInvalidRequestToken() bool {
	return e.Code == 10001
}

type response struct {
	*APIError
	Data interface{} `json:"data,omitempty"`
}

// Member 表示成员信息，对于同一个开发者帐号，同一个用户，无论在哪个您拥有的频道下，Member.ID 是一样的，您可以使用此 ID 作为关联的依据
type Member struct {
	ChannelCreator bool   `json:"channelCreator"`
	ExpiredAt      int64  `json:"expiredAt"`
	OpenID         string `json:"openID"`
}

// 推送消息主体
type Message struct {
	// 消息 ID，新消息留空，更新旧消息时，传入对应的消息 ID
	ID int64 `json:"id"`

	// 如果指定了接收人，则消息只会发给指定的人员
	Recipients []string `json:"recipients"`
	// 发给频道全体成员，则 Recipients 留空，同时设置 ToAll 为 true
	ToAll bool `json:"toAll"`

	// 模板 ID 和版本号，从后台模板管理中获取
	TemplateID      string `json:"templateID"`
	TemplateVersion int32  `json:"templateVersion"`

	// 消息标题，简要说明此消息内容，标题用于在客户端频道列表以及操作系统通知中显示
	Title string `json:"title"`

	// 消息内容/状态，可以为空
	Data map[string]interface{} `json:"data"`
}
