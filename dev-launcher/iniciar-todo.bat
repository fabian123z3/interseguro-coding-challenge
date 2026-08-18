@echo off
setlocal EnableExtensions

for %%I in ("%~dp0..") do set "RAIZ=%%~fI"

rem Las variables se definen una sola vez y los procesos hijos las heredan.
rem La forma set "NOMBRE=valor" evita espacios accidentales al final.
set "JWT_SECRET=secreto-local"
set "DEMO_USERNAME=demo"
set "DEMO_PASSWORD=demo1234"
set "STATS_API_URL=http://localhost:3000"

if /I "%~1"=="--check" goto comprobar

call :validar_herramientas
if errorlevel 1 goto fallo

call :asegurar_dependencias
if errorlevel 1 goto fallo

echo.
echo Levantando API Node ^(:3000^), API Go ^(:8080^) y Frontend ^(:5173^)...

where wt.exe >nul 2>&1
if errorlevel 1 goto ventanas_separadas

echo Abriendo Windows Terminal con tres pestanas...
wt.exe -w new new-tab --title "IS API Node" --startingDirectory "%RAIZ%\api-node" cmd.exe /k "npm run dev" ^; new-tab --title "IS API Go" --startingDirectory "%RAIZ%\api-go" cmd.exe /k "go run ./cmd/server" ^; new-tab --title "IS Frontend" --startingDirectory "%RAIZ%\frontend" cmd.exe /k "npm run dev"
if errorlevel 1 goto fallo
goto iniciado

:ventanas_separadas
echo Windows Terminal no esta disponible. Abriendo tres ventanas independientes...
start "IS API Node" /D "%RAIZ%\api-node" cmd.exe /k "npm run dev"
start "IS API Go" /D "%RAIZ%\api-go" cmd.exe /k "go run ./cmd/server"
start "IS Frontend" /D "%RAIZ%\frontend" cmd.exe /k "npm run dev"
goto iniciado

:iniciado
echo.
echo Servicios iniciados:
echo   Frontend:  http://localhost:5173
echo   API Go:    http://localhost:8080
echo   API Node:  http://localhost:3000
echo   Usuario:   demo
echo   Contrasena: demo1234
echo.
echo Cierra las pestanas o ventanas para detener los servicios.
pause
exit /b 0

:validar_herramientas
if not exist "%RAIZ%\api-node\package.json" (
    echo ERROR: no se encontro api-node\package.json en "%RAIZ%".
    exit /b 1
)
if not exist "%RAIZ%\api-go\go.mod" (
    echo ERROR: no se encontro api-go\go.mod en "%RAIZ%".
    exit /b 1
)
if not exist "%RAIZ%\frontend\package.json" (
    echo ERROR: no se encontro frontend\package.json en "%RAIZ%".
    exit /b 1
)
where npm.cmd >nul 2>&1
if errorlevel 1 (
    echo ERROR: npm no esta disponible en PATH. Instala Node.js 22 o superior.
    exit /b 1
)
where go.exe >nul 2>&1
if errorlevel 1 (
    if exist "%ProgramFiles%\Go\bin\go.exe" (
        set "PATH=%ProgramFiles%\Go\bin;%PATH%"
    ) else (
        echo ERROR: Go no esta disponible en PATH ni en "%ProgramFiles%\Go\bin".
        exit /b 1
    )
)
exit /b 0

:asegurar_dependencias
if not exist "%RAIZ%\api-node\node_modules" (
    echo Instalando dependencias de API Node...
    pushd "%RAIZ%\api-node"
    call npm install
    if errorlevel 1 (
        popd
        exit /b 1
    )
    popd
)

if not exist "%RAIZ%\frontend\node_modules" (
    echo Instalando dependencias del frontend...
    pushd "%RAIZ%\frontend"
    call npm install
    if errorlevel 1 (
        popd
        exit /b 1
    )
    popd
)
exit /b 0

:comprobar
echo Verificando launcher...
call :validar_herramientas
if errorlevel 1 goto fallo
echo RAIZ=[%RAIZ%]
echo JWT_SECRET=[%JWT_SECRET%]
echo DEMO_USERNAME=[%DEMO_USERNAME%]
echo DEMO_PASSWORD=[%DEMO_PASSWORD%]
echo STATS_API_URL=[%STATS_API_URL%]
echo Lanzador OK. No se inicio ningun proceso.
exit /b 0

:fallo
echo.
echo No se pudieron iniciar los servicios. Revisa el error mostrado arriba.
exit /b 1
