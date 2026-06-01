# Additional clean files
cmake_minimum_required(VERSION 3.16)

if("${CONFIG}" STREQUAL "" OR "${CONFIG}" STREQUAL "Release")
  file(REMOVE_RECURSE
  "CMakeFiles\\CampusAutoLogin_autogen.dir\\AutogenUsed.txt"
  "CMakeFiles\\CampusAutoLogin_autogen.dir\\ParseCache.txt"
  "CMakeFiles\\core_autogen.dir\\AutogenUsed.txt"
  "CMakeFiles\\core_autogen.dir\\ParseCache.txt"
  "CMakeFiles\\test_ip_detector_autogen.dir\\AutogenUsed.txt"
  "CMakeFiles\\test_ip_detector_autogen.dir\\ParseCache.txt"
  "CMakeFiles\\test_login_manager_autogen.dir\\AutogenUsed.txt"
  "CMakeFiles\\test_login_manager_autogen.dir\\ParseCache.txt"
  "CampusAutoLogin_autogen"
  "core_autogen"
  "test_ip_detector_autogen"
  "test_login_manager_autogen"
  )
endif()
