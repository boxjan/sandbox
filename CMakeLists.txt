cmake_minimum_required(VERSION 3.0)
project(sandbox)

set(CMAKE_CXX_STANDARD 14)

add_executable(sandbox src/main.cpp src/log.h)

add_test(test test/)