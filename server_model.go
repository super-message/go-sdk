package go_sdk

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// QueryParameter 代表用户通过客户端直接向开发者服务器发起请求时 url query 所带的参数
type QueryParameter struct {
	RequestToken string
	// 请求 token 的过期时间戳，用于缓存 token 时设置过期时间，不用每次都向平台发起请求验证 token，避免用户长时间等待
	TokenExpiredAt int64
	// 请求是从哪个频道、那条消息、使用哪个版本的模板发起的
	// 需要注意的是，MessageID 可能是 0，比如用户从菜单中触发的，这将在客户端中生成一条新的消息（平台不会保存）
	ChannelID       string
	MessageID       int64
	MessageLocalID  int64
	TemplateID      string
	TemplateVersion int
}

// QueryParameterFrom 是个辅助函数，用于从 http.Request.URL.Query 中获取请求参数
func QueryParameterFrom(r *http.Request) (q *QueryParameter, err error) {
	query := r.URL.Query()

	q = &QueryParameter{}

	// rt/rte/cid 是一定有的值
	q.RequestToken = strings.TrimSpace(query.Get("rt"))
	if q.RequestToken == "" {
		return nil, errors.New("request token is required")
	}

	tokenExpiredAt := query.Get("rte")
	if tokenExpiredAt == "" {
		return nil, errors.New("token expiration is required")
	}
	q.TokenExpiredAt, err = strconv.ParseInt(tokenExpiredAt, 10, 64)
	if err != nil {
		return nil, errors.New("invalid rte value")
	}

	q.ChannelID = strings.TrimSpace(query.Get("cid"))
	if q.ChannelID == "" {
		return nil, errors.New("channel id not presented")
	}

	// 下面的参数有传则按类型转换，无传则不管。按需调用其它方法进行检查
	id := query.Get("id")
	if id != "" {
		q.MessageID, err = strconv.ParseInt(id, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	localID := query.Get("lid")
	if localID != "" {
		q.MessageLocalID, err = strconv.ParseInt(localID, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	q.TemplateID = strings.TrimSpace(query.Get("tid"))

	templateVersion := query.Get("tv")
	if templateVersion != "" {
		q.TemplateVersion, err = strconv.Atoi(templateVersion)
		if err != nil {
			return nil, errors.New("invalid tv value")
		}
	}

	return
}

// CheckForMessageRequest 检查从【消息卡片】操作时发起的请求中，必传参数是否有效
func (q *QueryParameter) CheckForMessageRequest() error {
	if q.MessageID < 0 {
		return errors.New("invalid message id")
	}

	if q.MessageLocalID < 0 {
		return errors.New("invalid local message id")
	}

	if q.TemplateID == "" {
		return errors.New("template id is required")
	}

	if q.TemplateVersion <= 0 {
		return errors.New("invalid template version number")
	}

	return nil
}

type TipType int

const (
	Info    TipType = 0
	Success TipType = 1
	Warning         = 2
	Error           = 3
)

type Dismiss struct {
	Type     TipType `json:"type"`
	Duration int     `json:"duration"`
	Tip      string  `json:"tip"`
}

type DeleteMessage struct {
	ID      int64 `json:"id,omitempty"`
	LocalID int64 `json:"localID,omitempty"`
}

type UpdateMessage struct {
	ID              int64       `json:"id,omitempty"`
	LocalID         int64       `json:"localID,omitempty"`
	Title           string      `json:"title,omitempty"`
	Data            interface{} `json:"data,omitempty"`
	TemplateID      string      `json:"tid,omitempty"`
	TemplateVersion int         `json:"tv,omitempty"`
}

type NewMessage struct {
	TemplateID      string      `json:"tid,omitempty"`
	TemplateVersion int         `json:"tv,omitempty"`
	Title           string      `json:"title,omitempty"`
	Data            interface{} `json:"data,omitempty"`
}

type Response struct {
	Delete     *DeleteMessage `json:"delete,omitempty"`
	UpdatePart *UpdatePart    `json:"updatePart,omitempty"`
	Update     *UpdateMessage `json:"update,omitempty"`
	New        *NewMessage    `json:"new,omitempty"`
	Dismiss    *Dismiss       `json:"dismiss,omitempty"`
	Version    int            `json:"version"`
}

func NewResponse() *Response {
	return &Response{}
}

// DeleteThisMessage 删除消息，请求从哪条消息触发的则删除哪条消息，
// 如果需要删除其它消息，请自行填充相关字段
func (m *Response) DeleteThisMessage(q *QueryParameter) *Response {
	m.Delete = &DeleteMessage{
		ID:      q.MessageID,
		LocalID: q.MessageLocalID,
	}

	return m
}

func (m *Response) UpdatePartData(updatePart *UpdatePart) *Response {
	m.UpdatePart = updatePart
	return m
}

// UpdateThisMessage 更新消息，请求从哪条消息触发的则更新哪条消息，只更新 title、data
// 如果需要更改模板或其它消息，请自行填充相关字段
func (m *Response) UpdateThisMessage(q *QueryParameter, title string, data interface{}) *Response {
	m.UpdateThisMessageWithTemplate(q, q.TemplateID, q.TemplateVersion, title, data)
	return m
}

// UpdateThisMessageWithTemplate 更新消息和模板，请求从哪条消息触发的则更新哪条消息
func (m *Response) UpdateThisMessageWithTemplate(q *QueryParameter, templateID string, templateVersion int, title string, data interface{}) *Response {
	m.Update = &UpdateMessage{
		ID:              q.MessageID,
		LocalID:         q.MessageLocalID,
		Title:           title,
		Data:            data,
		TemplateID:      templateID,
		TemplateVersion: templateVersion,
	}

	return m
}

const DismissDuration = 1500

func (m *Response) ShowInfo(tip string) *Response {
	m.ShowTip(Info, tip, DismissDuration)
	return m
}

func (m *Response) ShowSuccess(tip string) *Response {
	m.ShowTip(Success, tip, DismissDuration)
	return m
}

func (m *Response) ShowWarning(tip string) *Response {
	m.ShowTip(Warning, tip, DismissDuration)
	return m
}

func (m *Response) ShowError(tip string) *Response {
	m.ShowTip(Error, tip, DismissDuration)
	return m
}

func (m *Response) ShowTip(t TipType, tip string, duration int) *Response {
	m.Dismiss = &Dismiss{
		Type:     t,
		Duration: duration,
		Tip:      tip,
	}
	return m
}

// Output 将数据编码输出，此后不能再输出其它内容
func (m *Response) Output(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(m)
}

// ShowInfo 是 *Response.ShowInfo 方法快捷方式，用于直接输出一个 Dismiss 普通提示信息，
// 不会有其它操作，并且此后不能再输出其它内容
func ShowInfo(w http.ResponseWriter, tip string) error {
	response := &Response{}
	return response.ShowInfo(tip).Output(w)
}

// ShowSuccess 是 *Response.ShowSuccess 方法快捷方式，用于直接输出一个 Dismiss 成功提示信息，
// 不会有其它操作，并且此后不能再输出其它内容
func ShowSuccess(w http.ResponseWriter, tip string) error {
	response := &Response{}
	return response.ShowSuccess(tip).Output(w)
}

// ShowWarning 是 *Response.ShowWarning 方法快捷方式，用于直接输出一个 Dismiss 警告提示信息，
// 不会有其它操作，并且此后不能再输出其它内容
func ShowWarning(w http.ResponseWriter, tip string) error {
	response := &Response{}
	return response.ShowWarning(tip).Output(w)
}

// ShowError 是 *Response.ShowError 方法快捷方式，用于直接输出一个 Dismiss 错误提示信息，
// 不会有其它操作，并且此后不能再输出其它内容
func ShowError(w http.ResponseWriter, tip string) error {
	response := &Response{}
	return response.ShowError(tip).Output(w)
}
