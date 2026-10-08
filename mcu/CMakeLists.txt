# Check which PlatformIO environment is currently building
if(NOT DEFINED ENV{PIOENV})
    set(CURRENT_ENV "control-node") # Fallback value if built outside PlatformIO
else()
    set(CURRENT_ENV $ENV{PIOENV})
endif()

# Apply source file filtering based on the active environment
if(CURRENT_ENV STREQUAL "control-node")
    # Equivalent to: +<control-node/> -<base-station/>
    idf_component_register(
        SRC_DIRS "control-node"
        INCLUDE_DIRS "control-node"
    )
else()
    # Equivalent to: +<base-station/> -<control-node/>
    idf_component_register(
        SRC_DIRS "base-station"
        INCLUDE_DIRS "base-station"
    )
endif()
