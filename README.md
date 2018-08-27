# Websocket log print
> 从kafka中取出K-V形式的json格式日志，打印到web上

## server
> 提供监听`log`开头的topic的`websocket`通讯，将收到的消息格式化后发送到`websocket`客户端

## wrapper
> 将和`server`的`websocket`通讯的控制部分封装成可与`JavaScript`原生`websocket`进行通讯

## ui
> 日志打印页面
