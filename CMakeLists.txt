cmake_minimum_required(VERSION 3.0)
project(sandbox)

#set(CMAKE_VERBOSE_MAKEFILE ON)

add_compile_options(-pie)
#add_compile_options(-O2)
add_compile_options(-Wall)
set(CMAKE_CXX_STANDARD 11)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_executable(sandbox.so src/main.cpp src/runner.cpp src/child.cpp src/runner.h src/child.h src/log.cpp)

target_link_libraries(sandbox.so seccomp pthread)

add_test(test test/)