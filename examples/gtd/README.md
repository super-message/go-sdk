#### GTD

一个简单的 GTD (Get Thing Done) 例子，展示如何如果与服务器和客户端交互。   
例子为了方便阅读和运行，只使用内存来存储数据。

代码中用到的模板如下：

**新增待办**

```html
<Input label="输入待办事项" name="title"></Input>
<Button api:post="/todo">添加进待办列表</Button>
```

**待办列表**

```html
<For list="list" item="todo" v:show="list">
    <CheckBox value="{{todo.id}}" name="list[]">{{todo.title}}</CheckBox>
</For>
<Text fontSize="18" color="#000" v:hide="list">当前待办列表很干净，可以</Text>
<Button api:post="/todos" type="primary">提交已完成事项</Button>
<Button api:get="/todos">刷新待办列表</Button>
```
