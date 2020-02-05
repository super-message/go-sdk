package go_sdk

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

var apiHost = "https://api.super-message.com"

func init() {
	if envHost, ok := os.LookupEnv("SM_API"); ok {
		apiHost = strings.TrimSpace(envHost)
	}
}

// RequestTokenCache 定义了一套用来缓存 request token 的接口
type RequestTokenCache interface {
	Get(rt string) (member Member, exist bool)
	// member.ExpiredAt 表示缓存过期时间那一刻的 UNIX 时间戳
	Set(rt string, member Member) error
	Delete(rt string)
}

// MemoryCache 实现了基于内存的 RequestTokenCache 接口
type MemoryCache struct {
	cache *cache.Cache
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (c *MemoryCache) Get(rt string) (member Member, exist bool) {
	v, exist := c.cache.Get(rt)
	if !exist {
		return
	}

	member = v.(Member)
	return
}

func (c *MemoryCache) Set(rt string, member Member) error {
	d := time.Second * time.Duration(member.ExpiredAt-time.Now().Unix())
	c.cache.Set(rt, member, d)
	return nil
}

func (c *MemoryCache) Delete(rt string) {
	c.cache.Delete(rt)
}

// Client 构建了几个与平台服务端接口进行交互的方法
type Client struct {
	pathPrefix  string
	accessToken string
	cache       RequestTokenCache
}

// NewClient 新建一个 Client 实例，其中 accessToken 为 Channel 访问平台接口的 token，
// cache 用于缓存 request token，避免每次都去调用平台接口验证，提升响应速度，提高用户体验。
// SDK 包构建了一个简单的基于内存的缓存，可以直接使用，另外也可以自行基于 Redis 构建一个持久
// 缓存，避免重启服务的时候又要从平台接口验证。
//
// 使用内存缓存：
//      client := NewClient("accessToken", NewMemoryCache())
// 不使用缓存：
//      client := NewClient("accessToken", nil)
func NewClient(accessToken string, cache RequestTokenCache) *Client {
	return &Client{
		pathPrefix:  "/v1",
		accessToken: accessToken,
		cache:       cache,
	}
}

var (
	ErrAccessTokenRequired    = errors.New("access token is required")
	ErrTemplateIDRequired     = errors.New("template id is required")
	ErrInvalidTemplateVersion = errors.New("invalid template version, version number must be greater than 0")
	ErrMessageTitleRequired   = errors.New("message title is required")
	ErrMessageIDRequired      = errors.New("message id is required")
)

func (c *Client) apiURL(path string) string {
	fmt.Println(apiHost + c.pathPrefix + path)
	return apiHost + c.pathPrefix + path
}

func (c *Client) doRequest(req *http.Request, expected interface{}) error {
	if c.accessToken == "" {
		return ErrAccessTokenRequired
	}

	req.URL.RawQuery += "&accessToken=" + c.accessToken
	apiResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if apiResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("the server api responses an unexpected status code, expect 200, but got %d", apiResponse.StatusCode)
	}

	rs := &response{
		Data: expected,
	}

	decoder := json.NewDecoder(apiResponse.Body)
	if err = decoder.Decode(rs); err != nil {
		return err
	}
	_ = apiResponse.Body.Close()

	if rs.Code != 0 {
		return rs.APIError
	}

	return nil
}

func (c *Client) Cache() RequestTokenCache {
	return c.cache
}

// VerifyRequestToken 用于验证客户端的请求 token 是否合法，如果 token 是合法的，则返回对应
// 成员（即哪个用户在客户端操作）的信息。如果 err != nil，并且你需要具体的错误情况时，
// 则执行 re, ok := err.(*APIError) 进行断言：
//      1. 当断言通过，则表示这通常是一个业务性的错误，通过 re.Code 和 re.Message 获得详细的
//          错误信息
//      2. 当断言失败，则表示错误通常是非业务性的或者本地发送的错误，比如网络问题，服务器故障、本
//      地数据不合法等，只能通过err.Error() 查看具体的信息
// 注意：
//      requestToken 有一个生命周期，在一个周期内，同个用户的操作请求中 requestToken 保持不变
//      ，建议缓存此数据
func (c *Client) VerifyRequestToken(requestToken string) (m Member, err error) {
	if c.cache != nil {
		m, exist := c.cache.Get(requestToken)
		if exist {
			return m, nil
		}
	}

	req, err := http.NewRequest("GET", c.apiURL("/user/verify")+"?token="+requestToken, nil)
	if err != nil {
		return
	}

	expected := &Member{}
	err = c.doRequest(req, expected)
	if err != nil {
		return
	}

	m = *expected
	if m.OpenID != "" && m.ExpiredAt-10 > time.Now().Unix() {
		_ = c.cache.Set(requestToken, m)
	}
	return
}

type MessageContentRequest struct {
	// 模板 ID 和版本号，从后台模板管理中获取
	TemplateID      string `json:"templateID"`
	TemplateVersion int32  `json:"templateVersion"`

	// 消息标题，简要说明此消息内容，标题用于在客户端频道列表以及操作系统通知中显示
	Title string `json:"title"`

	// 消息内容/状态，可以为空
	Data map[string]interface{} `json:"data"`
}

func (m *MessageContentRequest) check() error {
	m.TemplateID = strings.TrimSpace(m.TemplateID)
	if m.TemplateID == "" {
		return ErrTemplateIDRequired
	}

	if m.TemplateVersion < 1 {
		return ErrInvalidTemplateVersion
	}

	m.Title = strings.TrimSpace(m.Title)
	if m.Title == "" {
		return ErrMessageTitleRequired
	}

	return nil
}

type CreateMessageRequest struct {
	// 如果指定了接收人，则消息只会发给指定的人员
	Recipients []string `json:"recipients"`
	// 发给频道全体成员，则 Recipients 留空，同时设置 ToAll 为 true
	ToAll bool `json:"toAll"`
	MessageContentRequest
}

type createMessageResponse struct {
	ID int64 `json:"id"`
}

// CreateMessage 通过平台向频道或指定用户推送消息
// err 参考 VerifyRequestToken 接口 error 的处理方法
func (c *Client) CreateMessage(cmr *CreateMessageRequest) (messageID int64, err error) {
	if err := cmr.check(); err != nil {
		return 0, err
	}

	body := new(bytes.Buffer)
	encoder := json.NewEncoder(body)
	err = encoder.Encode(cmr)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", c.apiURL("/messages"), body)
	if err != nil {
		return
	}

	expected := &createMessageResponse{}
	err = c.doRequest(req, expected)
	if err != nil {
		return
	}

	messageID = expected.ID
	return
}

type UpdateMessageRequest struct {
	ID int64
	MessageContentRequest
}

// UpdateMessage 更新已有消息的内容，包括模版、标题和数据，不能原消息修改接收人
func (c *Client) UpdateMessage(umr *UpdateMessageRequest) (err error) {
	if umr.ID <= 0 {
		return ErrMessageIDRequired
	}

	if err := umr.MessageContentRequest.check(); err != nil {
		return err
	}

	body := new(bytes.Buffer)
	encoder := json.NewEncoder(body)
	err = encoder.Encode(umr)
	if err != nil {
		return
	}

	req, err := http.NewRequest("PUT", c.apiURL("/messages"), body)
	if err != nil {
		return
	}

	err = c.doRequest(req, nil)
	if err != nil {
		return
	}

	return
}

// DeleteMessage 删除一条已有的消息
func (c *Client) DeleteMessage(messageID int64) (err error) {
	if messageID <= 0 {
		return ErrMessageIDRequired
	}

	url := c.apiURL("/messages") + "?id=" + strconv.FormatInt(messageID, 10)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return
	}

	err = c.doRequest(req, nil)
	if err != nil {
		return
	}

	return
}
