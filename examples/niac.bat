@echo off
REM NIAC Demo Manager (Windows)
REM Unified command-line interface for Network-in-a-Can demos
REM Author: Kris Armstrong <kris.armstrong@me.com>

setlocal enabledelayedexpansion

set SCRIPT_DIR=%~dp0
set SCENARIOS_DIR=%SCRIPT_DIR%scenario_configs
set WALKS_DIR=%SCRIPT_DIR%device_walks
set CAPTURES_DIR=%SCRIPT_DIR%captures

REM Determine Java command
if defined JAVA_HOME (
    set JAVA_CMD=%JAVA_HOME%\bin\java.exe
) else (
    set JAVA_CMD=java
)

REM Main JAR location (check multiple possible locations)
set NIAC_JAR=%SCRIPT_DIR%..\lib\niac-6.0.jar
if not exist "!NIAC_JAR!" set NIAC_JAR=%SCRIPT_DIR%..\..\target\niac-6.0.jar
if not exist "!NIAC_JAR!" set NIAC_JAR=%SCRIPT_DIR%..\build\network_in_a_can.jar

REM Parse command
if "%~1"=="" goto show_usage
if "%~1"=="help" goto show_usage
if "%~1"=="--help" goto show_usage
if "%~1"="-h" goto show_usage
if "%~1"=="list" goto cmd_list
if "%~1"=="ls" goto cmd_list
if "%~1"=="run" goto cmd_run
if "%~1"=="start" goto cmd_run
if "%~1"=="walk" goto cmd_walk
if "%~1"=="walks" goto cmd_walk
if "%~1"=="device" goto cmd_walk
if "%~1"=="devices" goto cmd_walk

echo Error: Unknown command: %~1
echo.
goto show_usage

:show_usage
echo NIAC Demo Manager - Network-in-a-Can Demo Interface
echo.
echo Usage: %~nx0 ^<command^> [options]
echo.
echo Commands:
echo     list [scenarios^|walks^|captures]    List available resources
echo     run ^<scenario^>                     Run a demo scenario
echo     walk ^<vendor^> [device]             Show available device walks
echo     help                               Show this help message
echo.
echo Examples:
echo     %~nx0 list scenarios    # List all available demo scenarios
echo     %~nx0 list walks        # List all device vendor folders
echo     %~nx0 run nexus         # Run the Nexus demo scenario
echo     %~nx0 walk cisco        # List all Cisco device walks
echo     %~nx0 walk dell         # List all Dell device walks
echo.
echo Available Scenarios:
if exist "%SCENARIOS_DIR%" (
    for %%f in ("%SCENARIOS_DIR%\*.cfg") do (
        set "scenario=%%~nf"
        echo     - !scenario!
    )
)
echo.
echo Available Device Vendors:
if exist "%WALKS_DIR%" (
    for /d %%d in ("%WALKS_DIR%\*") do (
        set "vendor=%%~nxd"
        set count=0
        for %%w in ("%%d\*.walk") do set /a count+=1
        echo     - !vendor! ^(!count! walks^)
    )
)
goto :eof

:cmd_list
set type=%~2
if "!type!"=="" set type=scenarios

if "!type!"=="scenarios" goto list_scenarios
if "!type!"=="scenario" goto list_scenarios
if "!type!"=="walks" goto list_walks
if "!type!"=="walk" goto list_walks
if "!type!"=="devices" goto list_walks
if "!type!"=="captures" goto list_captures
if "!type!"=="capture" goto list_captures
if "!type!"=="caps" goto list_captures

echo Error: Unknown list type: !type!
echo Valid types: scenarios, walks, captures
goto :eof

:list_scenarios
echo Available Demo Scenarios:
if exist "%SCENARIOS_DIR%" (
    for %%f in ("%SCENARIOS_DIR%\*.cfg") do (
        set "scenario=%%~nf"
        echo   - !scenario!
    )
) else (
    echo Error: Scenarios directory not found: %SCENARIOS_DIR%
)
goto :eof

:list_walks
echo Available Device Vendors:
if exist "%WALKS_DIR%" (
    for /d %%d in ("%WALKS_DIR%\*") do (
        set "vendor=%%~nxd"
        set count=0
        for %%w in ("%%d\*.walk") do set /a count+=1
        echo   !vendor!                 ^(!count! walks^)
    )
) else (
    echo Error: Walks directory not found: %WALKS_DIR%
)
goto :eof

:list_captures
echo Available Packet Captures:
if exist "%CAPTURES_DIR%" (
    for %%f in ("%CAPTURES_DIR%\*.pcap" "%CAPTURES_DIR%\*.cap") do (
        if exist "%%f" (
            set "capture=%%~nxf"
            echo   - !capture!
        )
    )
) else (
    echo Error: Captures directory not found: %CAPTURES_DIR%
)
goto :eof

:cmd_run
set scenario=%~2

if "!scenario!"=="" (
    echo Error: Scenario name required
    echo Usage: %~nx0 run ^<scenario^>
    echo.
    goto list_scenarios
)

set cfg_file=%SCENARIOS_DIR%\!scenario!.cfg

if not exist "!cfg_file!" (
    echo Error: Scenario not found: !scenario!
    echo.
    echo Available scenarios:
    goto list_scenarios
)

if not exist "!NIAC_JAR!" (
    echo Error: NIAC JAR not found: !NIAC_JAR!
    echo Please build the project first
    goto :eof
)

echo Starting scenario: !scenario!
echo Config file: !cfg_file!
echo.

REM Run the NIAC with the configuration
"!JAVA_CMD!" -jar "!NIAC_JAR!" -c "!cfg_file!"
goto :eof

:cmd_walk
set vendor=%~2
set device=%~3

if "!vendor!"=="" (
    echo Error: Vendor name required
    echo Usage: %~nx0 walk ^<vendor^> [device]
    echo.
    goto list_walks
)

set vendor_dir=%WALKS_DIR%\!vendor!

if not exist "!vendor_dir!" (
    echo Error: Vendor not found: !vendor!
    echo.
    echo Available vendors:
    goto list_walks
)

if "!device!"=="" (
    echo Available walks for !vendor!:
    for %%f in ("!vendor_dir!\*.walk") do (
        set "name=%%~nf"
        echo   - !name!
    )
) else (
    set walk_file=!vendor_dir!\!vendor!-!device!.walk
    if not exist "!walk_file!" (
        echo Error: Device walk not found: !walk_file!
        echo.
        echo Available walks for !vendor!:
        for %%f in ("!vendor_dir!\*.walk") do (
            set "name=%%~nf"
            echo   - !name!
        )
        goto :eof
    )

    echo Device Walk: !walk_file!
    echo.
    type "!walk_file!"
)
goto :eof

endlocal
