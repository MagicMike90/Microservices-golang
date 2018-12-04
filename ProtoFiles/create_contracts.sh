#!/bin/bash

protoc --go_out=plugins=grpc:. *.proto
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. user_data.proto