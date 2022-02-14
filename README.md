# **Consistent Hashing Demo**

A simple demo of consistent hashing.

<br/>

## **Features**

These features have been implemented:

- Core consistent-hashing-algorithm
- Consistent Hashing with Bounded Loads, according to [Consistent Hashing with Bounded Loads](http://ai.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html)
- Customize hash function support
- Customize replica number on hash ring support
- A simple load-balance proxy demo
- A full demo to show how consistent-hashing-algorithm works

<br/>

## **How to use**

### **Step 1: Start Server**

First, start the proxy server:

```shell
$ go run main.go

start proxy server: 18888
```

This will start the proxy server on port 18888.

The server will listen for incoming registration and will forward key searching to the servers.

Then, start the k-v server:

```shell
$ go run server/main.go

start server: 8080
```

Also, the server will register to the proxy server:

```
register host: localhost:8080 success
```

> **Notice: you can register multiple servers to the proxy server:**
>
> ```shell
> go run server/main.go -p 8081
> go run server/main.go -p 8082
> ……
> ```

<br/>

### **Step 2: Query**

Use `curl` to get the key from proxy：

```shell
$ curl localhost:18888/key?key=123

key: 123, val: hello: 123
```

If you query the key at the first time，the key will be cached on the corresponding server for 10s.

The log for proxy server:

```
Response from host localhost:8080: hello: 123
```

The log for k-v server:

```
cached key: {123: hello: 123}
removed cached key after 3s: {123: hello: 123}
```

<br/>

### **Step 3: Try Query Different Key**

You can try to query different key, to check whether they are cached on the different servers:

```
Response from host localhost:8082: hello: 45363456
Response from host localhost:8080: hello: 4
Response from host localhost:8082: hello: 1
Response from host localhost:8080: hello: 2
Response from host localhost:8082: hello: 3
Response from host localhost:8080: hello: 4
Response from host localhost:8082: hello: 5
Response from host localhost:8080: hello: 6
Response from host localhost:8082: hello: sdkbnfoerwtnbre
Response from host localhost:8082: hello: sd45555254tg423i5gvj4v5
Response from host localhost:8081: hello: 0
Response from host localhost:8082: hello: 032452345
```

<br/>

### **Step 4: Consistent Hash with Load Bound Test**

You can request for `localhost:18888/key_least` for Load Bound testing:

```bash
$ curl localhost:18888/key_least?key=123
key: 123, val: hello: 123
```

the result is shown below:

```
start proxy server: 18888
register host: localhost:8080 success
register host: localhost:8081 success
register host: localhost:8082 success

Response from host localhost:8080: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8081: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8081: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8081: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8081: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8080: hello: 123
Response from host localhost:8082: hello: 123
Response from host localhost:8082: hello: 123
```

You can see that, Consistent Hash with Load Bound is far more average!

>   You can change the `loadBoundFactor` for more experiments:
>
>   ```go
>   var (
>   	// the default number of replicas
>   	defaultReplicaNum = 10
>   
>   	// the load bound factor
>   	// ref: https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
>   	loadBoundFactor = 0.25
>   	......
>   )
>   ```

<br/>

## **Reference**

Reference:

-   https://segmentfault.com/a/1190000041268497
-   https://zh.wikipedia.org/wiki/%E4%B8%80%E8%87%B4%E5%93%88%E5%B8%8C
-   https://ai.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
-   https://pkg.go.dev/crypto/sha512#Sum512

Linked Blog:

-   [一致性Hash算法总结与应用](https://jasonkayzk.github.io/2022/02/12/一致性Hash算法总结与应用/)

