[![Go](https://github.com/Tracnac/orasql/actions/workflows/go.yml/badge.svg)](https://github.com/Tracnac/orasql/actions/workflows/go.yml) [![CodeQL](https://github.com/Tracnac/orasql/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Tracnac/orasql/actions/workflows/codeql-analysis.yml)

# orasql
An autonomous Oracle sql toolbox.

__Usage:__
```man
orasql  -dsn server_url -query sql
    -db string
        oracle by default not use yet
    -dsn string
        user:pass@dsn/service_name
        Env: ORASQL_DSN, ORASQL_USER, ORASQL_PWD
    -query string
        select 'column' as column_name from dual
    -debug
        Show column type (Default output only)
    -o  { csv, json, kv, xls or out (default) }
            csv   CSV Output
            json  JSON Output
            kv    Key/Value Output (2 columns max)
            xls   Excel file output
    -of string
            Output file (default "/dev/stdout")
    -i  { pipe, sql, json, dir }
            pipe Reqd from stdin
            sql  Read the query from file
            json Read options from file
    -if string
        File (default "/dev/stdin")

    -i work with -if
    -o work with -of

    By default:
     -o out
     -of /dev/sdtout
     -if /dev/sdtin
     

Example:

    ./orasql -dsn "oracle://user:pass@server/service_name" -query "select sysdate from dual"
    ./orasql -dsn "oracle://user:pass@server/service_name" -i sql -if query.sql
    ./orasql -i json -if sql.json
    echo 'select sysdate from dual' |  ./orasql -i pipe -dsn "oracle://user:pass@server/service_name"

With os environment: 
    ORACLESQL_DSN=127.0.0.1:1521/DB
    ORACLESQL_USER=user
    ORACLESQL_PWD=password
    orasql -query "select sysdate from dual"

default output:
    SYSDATE    : 2022-01-06 18:26:37 +0000 UTC

-debug:
    SYSDATE    [DATE]           : 2022-01-06 19:26:27 +0000 UTC

-o json:
    [
        {"SYSDATE": "2022-01-06T18:21:57Z"}
    ]

-o csv:
    "SYSDATE"
    "2022-01-06 18:28:03 +0000 UTC"

-o kv: with ("select 'Date', sysdate from dual"):
    "Date": "2022-01-06T19:21:21Z"

-i json -if sql.json:
      {
        "db": "oracle"
        "dsn": "127.0.0.1:1521/DB",
        "user": "user",
        "password": "password",
        "query": "select sysdate from dual"
      }
```
