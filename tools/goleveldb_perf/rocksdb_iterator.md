# RocksDB iterator memory consumption
This file describes the issue with RocksDB iterator memory usage as well as the steps to reproduce it.

## Issue
When iterating over RocksDB using the iterator, the process might consume more than 4GiB of additional virtual memory.

## Steps to reproduce
In `misbehavior_test.go` file there are two functions: _TestRocksDBWithoutIterator_ and _TestRocksDBWithIterator_. 
They both do the same, namely filling the RocksDB backend with random data, but _TestRocksDBWithIterator_ also iterates
the DB after each fill.

One can either monitor the process virtual memory usage using such tools as `top` or perform following steps:
* Set the virtual memory limit to 14GiB  
`ulimit -v 14000000`
* From the directory with `misbehavior_test.go` file  
`go test -tags "rocksdb" -run TestRocksDBWithIterator .`  
The test should fail due to the violation of memory constraint
* Set the virtual memory limit to 10GiB  
  `ulimit -v 10000000`
* From the directory with `misbehavior_test.go` file  
  `go test -tags "rocksdb" -run TestRocksDBWithoutIterator .`  
The test should pass
