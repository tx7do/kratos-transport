echo off

::指定起始文件夹
set DIR1="%cd%\..\_example\broker"
set DIR2="%cd%\..\_example\server"

for /R %DIR1% /d %%i in (*) do (
    echo %%i
    cd %%i
    go get all
    go mod tidy
)

for /R %DIR2% /d %%i in (*) do (
    echo %%i
    cd %%i
    go get all
    go mod tidy
)
