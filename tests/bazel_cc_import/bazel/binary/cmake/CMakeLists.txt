cmake_minimum_required(VERSION 3.16)
project(cmake_test C)

add_executable(cmake_test impl.c)
find_path(CMAKE_TEST_INCLUDE_DIR api.h REQUIRED)
find_library(CMAKE_TEST_STATIC_LIBRARY NAMES cmake_test_static REQUIRED)

target_include_directories(cmake_test PRIVATE ${CMAKE_TEST_INCLUDE_DIR})
target_link_libraries(cmake_test PRIVATE ${CMAKE_TEST_STATIC_LIBRARY})

install(TARGETS cmake_test RUNTIME DESTINATION bin)
