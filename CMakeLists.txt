cmake_minimum_required(VERSION 3.0)
project(sandbox)

#set(CMAKE_VERBOSE_MAKEFILE ON)

add_compile_options(-pie)
add_compile_options(-Wall)
set(CMAKE_CXX_STANDARD 11)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_executable(sandbox.so src/main.cpp src/child.cpp src/child.h src/watcher.cpp src/watcher.h)

target_link_libraries(sandbox.so seccomp)

add_test(test test/)