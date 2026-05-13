cmake_minimum_required(VERSION 3.16)
project(cmake_test C)

add_library(cmake_test_static STATIC impl.c)
add_library(cmake_test_shared SHARED impl.c)
target_include_directories(cmake_test_static PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})
target_include_directories(cmake_test_shared PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

install(TARGETS cmake_test_static ARCHIVE DESTINATION lib)
install(TARGETS cmake_test_shared LIBRARY DESTINATION lib RUNTIME DESTINATION bin)
install(FILES api.h DESTINATION include)
