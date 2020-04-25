package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	go_sdk "github.com/super-message/go-sdk"
)

var (
	client *go_sdk.Client

	todoStore = NewTodoStore()
)

func main() {
	// Access token 请到开发者后台查看
	client = go_sdk.NewClient("RCnXKNJW1AtjmA0Ih2xCAINrawzaM959", go_sdk.NewMemoryCache())

	router := mux.NewRouter()
	router.Methods("GET").Path("/todos").HandlerFunc(TodoList)
	router.Methods("POST").Path("/todo").HandlerFunc(AddTodo)
	router.Methods("GET").Path("/todo/done").HandlerFunc(TodoDone)
	// 频道订阅和退订钩子
	// 在用户订阅和退订时此钩子会被请求
	router.Methods("GET").Path("/hook/subscribe").HandlerFunc(SubscribeHook)
	router.Use(authMiddleware)
	http.Handle("/", router)

	log.Println("Starting server at :10086...")
	log.Fatal(http.ListenAndServe(":10086", router))
}

func authMiddleware(h http.Handler) http.Handler {
	return authMiddlewareHandler{originHandler: h}
}

type authMiddlewareHandler struct {
	originHandler http.Handler
}

const (
	contextKey = 1
)

type ContextValue struct {
	QueryParameter *go_sdk.QueryParameter
	Member         go_sdk.Member
}

func getContextValue(r *http.Request) ContextValue {
	return r.Context().Value(contextKey).(ContextValue)
}

// 集中处理请求认证
func (h authMiddlewareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q, err := go_sdk.QueryParameterFrom(r)
	if err != nil {
		log.Printf("unable to unmarshal request data: %s", err)
		// demo 忽略掉错误，如果发生了错误你应该记录下来
		_ = go_sdk.ShowError(w, "无法解析数据")
		return
	}

	member, err := client.VerifyRequestToken(q.RequestToken)
	if err != nil {
		if aerr, ok := err.(*go_sdk.APIError); ok {
			//  https://docs.super-message.com/api/server/#错误码列表
			switch aerr.Code {
			case 10000:
				fallthrough
			case 10001:
				_ = go_sdk.ShowError(w, "无法验证身份，request token 无效")

				return
			}
		}

		log.Println("failed to verify request token: ", err)
		_ = go_sdk.ShowError(w, "暂时无法为您提供服务")
		return
	}

	// 把解析过的参数和身份存进 context 里面，这样在具体的请求处理函数里面就可以直接拿来用了
	ctx := context.WithValue(r.Context(), contextKey, ContextValue{q, member})
	h.originHandler.ServeHTTP(w, r.WithContext(ctx))
}

type ResTodoList struct {
	List []*Todo `json:"list,omitempty"`
}

// 获取待办列表
func TodoList(w http.ResponseWriter, r *http.Request) {
	ctxval := getContextValue(r)

	_ = go_sdk.NewResponse().UpdateThisMessage(
		ctxval.QueryParameter,
		"待办列表",
		ResTodoList{
			List: todoStore.ListTodo(ctxval.Member.OpenID),
		}).
		Output(w)
}

func AddTodo(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("unable to read request data", err)
		_ = go_sdk.ShowError(w, "读取请求数据失败")
		return
	}
	defer r.Body.Close()

	p := &struct {
		Title string `json:"title"`
	}{}
	err = json.Unmarshal(b, p)
	if err != nil {
		log.Println("unable to parse request data", err)
		_ = go_sdk.ShowError(w, "解析请求数据失败")
		return
	}

	ctxval := getContextValue(r)
	todoStore.AddTodo(&Todo{
		UserID: ctxval.Member.OpenID,
		Title:  p.Title,
		Done:   false,
	})

	_ = go_sdk.NewResponse().
		DeleteThisMessage(ctxval.QueryParameter).
		ShowSuccess("任务已添加进代办列表").
		Output(w)
}

func TodoDone(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	index, err := strconv.Atoi(query.Get("i"))
	if err != nil {
		go_sdk.ShowError(w, "参数错误")
		return
	}

	id, err := strconv.Atoi(query.Get("id"))
	if err != nil {
		go_sdk.ShowError(w, "id 解析错误")
		return
	}

	ctxval := getContextValue(r)
	todoStore.DeleteTodo(id, ctxval.Member.OpenID)

	ops := go_sdk.NewUpdatePart()
	ops.AddOpRemove(go_sdk.NewRemove("list", []int{index}))

	_ = go_sdk.NewResponse().UpdatePartData(ops).Output(w)
}

func SubscribeHook(w http.ResponseWriter, r *http.Request) {
	ctxval := getContextValue(r)
	log.Println(ctxval.Member, "action", r.URL.Query().Get("_a"))

	action := r.URL.Query().Get("_a")
	if action == "subscribe" {
		// 订阅频道事件
		// doSomething()
	} else if action == "unsubscribe" {
		// 退订频道事件
		// doSomething()

		// 如果缓存了 request token，记得删除此 token
		client.Cache().Delete(r.URL.Query().Get("_rt"))
	}

	// 没有什么内容需要返回的，直接返回 204（2xx 都行）
	w.WriteHeader(http.StatusNoContent)
}
