# ancestor

## Prepare

### 1. Go
Install Go(v1.10+),you can see this page: [https://github.com/golang/go](https://github.com/golang/go)

### 2. Protobuf
Install google Protocol Buffer(v3.0.0+),you can get Protobuf in this page:
[https://github.com/google/protobuf](https://github.com/google/protobuf)

### 3. gogoprotobuf
To install it,you should first have Go and Protobuf installed.You can get gogoprotobuf in this page: [https://github.com/gogo/protobuf](https://github.com/gogo/protobuf)

### 4. MySQL client
We use MySQL client to test in this example.

Install MySQL,you can see this page: [http://dev.mysql.com/downloads/mysql/](http://dev.mysql.com/downloads/mysql/)

### 5.etcd(v3.0.0+)
Install etcd,you can see this page: [https://github.com/coreos/etcd](https://github.com/coreos/etcd)

## Install
    make
    
## Example
1.Build tool

    make tool

2.Enter the example directory:

    cd example/

3.Run cp.sh:

    sh cp.sh

4.Start etcd cluster:

    goreman start

5.Start at least two pd server:

    ./pdserver -c ../config/pdserver.cfg
    ./pdserver -c ./pd/pdserver.cfg

6.Init:

    ./tool -c ./../config/toolclient.cfg

7.Start at least two node:

    ./nodeserver -c 9001/nodeserver.cfg
    ./nodeserver -c 9002/nodeserver.cfg
    ./nodeserver -c 9003/nodeserver.cfg



