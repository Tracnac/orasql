[![Go](https://github.com/Tracnac/orasql/actions/workflows/go.yml/badge.svg)](https://github.com/Tracnac/orasql/actions/workflows/go.yml) [![CodeQL](https://github.com/Tracnac/orasql/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Tracnac/orasql/actions/workflows/codeql-analysis.yml)

# orasql
An autonomous Oracle sql toolbox.

__Usage:__
```man
orasql  -dsn server_url -query sql
    -db string
        oracle by default
    -dsn string
        user:pass@dsn/service_name
        Env: ORASQL_DSN, ORASQL_USER, ORASQL_PWD
    -query string
        select sysdate from dual
    -debug
        Show column type (Default output only)
    -o  { csv, json, yml, kv, xls or out (default) }
            csv   CSV Output
            json  JSON Output
            yml   YAML Output
            kv    Key/Value Output (2 columns max)
            xls   Excel file output
                  (if the xls file already exists a new sheet will be created)
    -of string
            Output file (default "/dev/stdout")
            xlsx[:sheetName] ("books.xlsx:My_Sheet" will create a Sheet named "My_Sheet" 
                              if the sheet already exists it will be deleted, so beware
                              of referenced values from this sheet to an another one)
    -i  { pipe, sql, json, yml, dir }
            pipe Read from stdin
            sql  Read the query from file
            json Read all parameters from JSON file
            yml  Read all parameters from YAML file
    -if string
        File (default "/dev/stdin")

    -i work with -if
    -o work with -of

    By default:
     -o out
     -of /dev/stdout
     -if /dev/stdin

Example:
    ./orasql  -db 'oracle' -dsn "user:pass@server/service_name" -query "select sysdate from dual"
    ./orasql  -dsn "user:pass@server/service_name" -i sql -if query.sql
    ./orasql  -i json -if sql.json
    ./orasql  -i yml -if sql.yml
    ./orasql -i json -if sql.json -o xls -of Excel.xlsx
    ./orasql -i json -if sql.json -o xls -of Excel.xlsx:sheet_name
    echo 'select sysdate from dual' |  ./orasql  -i pipe -dsn "oracle://user:pass@server/service_name"

With os environment: 
    ORACLESQL_DSN=127.0.0.1:1521/DB
    ORACLESQL_USER=user
    ORACLESQL_PWD=password
    orasql  -query "select sysdate from dual"

default output:
    SYSDATE    : 2022-01-06 18:26:37 +0000 UTC

-debug:
    SYSDATE    [DATE]           : 2022-01-06 19:26:27 +0000 UTC

-o json:
    [
        {"SYSDATE": "2022-01-06T18:21:57Z"}
    ]

-o yml:
    oraSQL:
      Lines:
        '0':
          -  SYSDATE: 2022-01-19T19:22:11Z
      LineCount: 1

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
      "pwd": "password",
      "query": "select sysdate from dual"
    }

-i yml -if sql.yml:
    db: "oracle"
    dsn: "127.0.0.1:1521/DB"
    user: user
    pwd: password
    query: "select sysdate from dual"
```

#### Work in progress
#### TODO:
- [ ] Append mode for out mode
- [x] Option for the sheet name in xls file
- [x] Input/Output YAML format
- [ ] Implement read queries from directory
- [ ] save/run/delete queries from sqlite
- [ ] Query array in json file
- [ ] Auto naming output file when multiple queries
- [ ] Run queries in parallel
- [ ] Gui
