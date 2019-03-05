cmake_minimum_required(VERSION 3.0)
project(sandbox)

set(GENERATE_NAME libsandbox.so)


add_compile_options(-pie)
add_compile_options(-O2)
add_compile_options(-Wall)
add_compile_options(-fPIC)
#add_compile_options(-g)
set(CMAKE_CXX_STANDARD 11)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_executable(${GENERATE_NAME} src/main.cpp src/runner.cpp src/child.cpp src/runner.h src/child.h src/log.cpp src/scmp/scmp.cpp src/scmp/scmp.h src/scmp/semp_compile.cpp src/scmp/scmp_strict.cpp src/scmp/scmp_gentle.cpp)

target_link_libraries(${GENERATE_NAME} seccomp pthread)

enable_testing()
add_test(test python3 test/run_test.py 0)
set_tests_properties(test PROPERTIES PASS_REGULAR_EXPRESSION "All is OK")



