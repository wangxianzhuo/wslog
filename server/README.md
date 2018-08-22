# Websocket server
> 从kafka中取出K-V形式大json格式日志，打印到web上

## Dependency

```shell
dep ensure
```

## Build

```shell
go build -o wslog main.go
```

或

```shell
./build.sh
```

## Usage

```shell
wslog --debug
```