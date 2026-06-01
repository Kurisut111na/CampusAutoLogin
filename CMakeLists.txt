cmake_minimum_required(VERSION 3.16)
project(CampusAutoLogin VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTOUIC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Network Widgets Test)

set(CORE_SOURCES
    src/core/logger.cpp
    src/core/ip_detector.cpp
    src/core/config_manager.cpp
    src/core/crypto_utils.cpp
    src/core/login_manager.cpp
    src/core/heartbeat.cpp
    src/core/network_manager.cpp
)

set(CORE_HEADERS
    src/core/logger.h
    src/core/ip_detector.h
    src/core/config_manager.h
    src/core/crypto_utils.h
    src/core/login_manager.h
    src/core/heartbeat.h
    src/core/network_manager.h
)

add_library(core STATIC ${CORE_SOURCES} ${CORE_HEADERS})
target_include_directories(core PUBLIC ${CMAKE_SOURCE_DIR}/src/core)
target_link_libraries(core PUBLIC Qt6::Core Qt6::Network crypt32)

set(UI_SOURCES
    src/ui/main_window.cpp
    src/ui/tray_icon.cpp
    src/ui/auto_start.cpp
    src/ui/advanced_settings_dialog.cpp
)

set(UI_HEADERS
    src/ui/main_window.h
    src/ui/tray_icon.h
    src/ui/auto_start.h
    src/ui/advanced_settings_dialog.h
)

add_executable(CampusAutoLogin
    src/main.cpp
    ${UI_SOURCES}
    ${UI_HEADERS}
)

target_link_libraries(CampusAutoLogin PRIVATE
    core
    Qt6::Core
    Qt6::Network
    Qt6::Widgets
)

target_include_directories(CampusAutoLogin PRIVATE ${CMAKE_SOURCE_DIR}/src/core)

set_target_properties(CampusAutoLogin PROPERTIES WIN32_EXECUTABLE TRUE)

set(TEST_SOURCES
    tests/test_ip_detector.cpp
    tests/test_login_manager.cpp
)

foreach(TEST_SRC ${TEST_SOURCES})
    get_filename_component(TEST_NAME ${TEST_SRC} NAME_WE)
    add_executable(${TEST_NAME} ${TEST_SRC})
    target_link_libraries(${TEST_NAME} PRIVATE core Qt6::Test)
    target_include_directories(${TEST_NAME} PRIVATE ${CMAKE_SOURCE_DIR}/src/core)
    add_test(NAME ${TEST_NAME} COMMAND ${TEST_NAME})
endforeach()