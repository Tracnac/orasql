# orasql
An autonomous Oracle sql toolbox


__Usage:__
```man
Usage:
./orasql  -dsn server_url -query sql
  -csv
    	CSV Output
  -debug
    	Show column type (Default output only)
  -dsn string
    	oracle://user:pass@dsn/service_name
    	Env: ORACLESQL_DSN, ORACLESQL_USER, ORACLESQL_PWD
  -file string
    	Input query from file (default "/dev/stdin")
  -json
    	JSON Output
  -kv
    	Key/Value Output (2 columns max)
  -output string
    	Output file (default "/dev/stdout")
  -payload string
    	Input payload Json from file
  -query string
    	select 'column' as column_name from dual

Example:

   ./orasql  -dsn "oracle://user:pass@server/service_name" -query "select sysdate from dual"
   ./orasql  -dsn "oracle://user:pass@server/service_name" -file query.sql
   ./orasql  -payload payload.json
   echo 'select sysdate from dual' |  ./orasql  -dsn "oracle://user:pass@server/service_name"

With os.env: 
   ORACLESQL_DSN=127.0.0.1:1521/DB ORACLESQL_USER=user ORACLESQL_PWD=password  ./orasql  -query "select sysdate from dual"

default output:
  SYSDATE    : 2022-01-06 18:26:37 +0000 UTC

-debug:
  SYSDATE    [DATE]           : 2022-01-06 19:26:27 +0000 UTC

-json:
  [
    {"SYSDATE": "2022-01-06T18:21:57Z"}
  ]

-csv:
  "SYSDATE"
  "2022-01-06 18:28:03 +0000 UTC"

-kv with ("select 'Date', sysdate from dual"):
  "Date": "2022-01-06T19:21:21Z"

-payload:
With json file:
  {
    "dsn": "127.0.0.1:1521/DB",
    "user": "user",
    "password": "password",
    "query": "select sysdate from dual"
  }
```
