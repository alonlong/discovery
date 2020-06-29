#!/bin/bash

etcd --name default \
 --listen-peer-urls="http://192.168.1.100:2380" \
 --listen-client-urls="http://192.168.1.100:2379,http://127.0.0.1:2379" \
 --advertise-client-urls="http://192.168.1.100:2379"