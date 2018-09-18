/*
Package engine readme doc:

1.Install Rocksdb's library
  * **Linux - Ubuntu**
    * Upgrade your gcc to version at least 4.7 to get C++11 support.
    * Install gflags. `sudo apt-get install libgflags-dev`
    * Install snappy. `sudo apt-get install libsnappy-dev`.
    * Install zlib. `sudo apt-get install zlib1g-dev`.
    * Install bzip2: `sudo apt-get install libbz2-dev`.
    * compiling RocksDB: `make shared_lib` will compile librocksdb.so, RocksDB shared library.

2.Install google protocbuf library 3.0.0:
    `sh ./autogen.sh`
    `./configure`
    `make & make install`

3.git clone https://github.com/gogo/protobuf at $GOPATH/src/github.com/gogo/

4. export env variables export LD_LIBRARY_PATH=$LD_LIBRARY:/usr/local/lib
*/
package engine
